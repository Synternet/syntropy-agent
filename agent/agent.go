package agent

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"github.com/SyntropyNet/syntropy-agent-go/config"
	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/SyntropyNet/syntropy-agent-go/controller/blockchain"
	"github.com/SyntropyNet/syntropy-agent-go/controller/saas"
	"github.com/SyntropyNet/syntropy-agent-go/controller/script"
	"github.com/SyntropyNet/syntropy-agent-go/logger"
	"github.com/SyntropyNet/syntropy-agent-go/netfilter"
	"github.com/SyntropyNet/syntropy-agent-go/pinger"
	"github.com/SyntropyNet/syntropy-agent-go/wireguard"
)

const pkgName = "SyntropyAgent. "

type Agent struct {
	running    uint32
	controller controller.Controller

	wg        *wireguard.Wireguard
	ping      *pinger.Pinger
	wgWatcher *WgPeerWatcher

	commands map[string]func(a *Agent, req []byte) error
}

// NewAgent allocates instance of agent struct
// Parses shell environment and setups internal variables
func NewAgent(contype int) (*Agent, error) {
	var err error
	agent := new(Agent)

	switch contype {
	case config.ControllerSaas:
		agent.controller, err = saas.NewController()
	case config.ControllerScript:
		agent.controller, err = script.NewController()
	case config.ControllerBlockchain:
		agent.controller, err = blockchain.NewController()
	default:
		err = fmt.Errorf("unexpected controller type %d", contype)
	}
	if err != nil {
		return nil, err
	}

	// Config loggers early - to get more info logged
	// TODO: do not spam controller in development stage
	// logger.SetControllerWriter(agent.controller)
	logger.Setup(config.GetDebugLevel(), os.Stdout)

	agent.wg, err = wireguard.New()
	if err != nil {
		return nil, err
	}

	agent.ping = pinger.NewPinger(agent)
	agent.wgWatcher = NewWgPeerWatcher(agent.wg, agent)

	agent.commands = make(map[string]func(a *Agent, req []byte) error)
	agent.commands["AUTO_PING"] = autoPing
	agent.commands["GET_INFO"] = getInfo
	agent.commands["CONFIG_INFO"] = configInfo
	agent.commands["WG_CONF"] = wireguardConfigure

	netfilter.CreateChain()

	return agent, nil
}

func (agent *Agent) messageHandler() {
	// Mark as not running on exit
	defer atomic.StoreUint32(&agent.running, 0)

	for {
		raw, err := agent.controller.Recv()

		if err == io.EOF {
			// Stop runner if the reader is done
			logger.Info().Println(pkgName, "Closing message handler - EOF")
			return
		} else if err != nil {
			// Simple errors are handled inside controller. This should be only fatal errors
			logger.Error().Println(pkgName, "Message handler error: ", err)
			return
		}

		agent.processCommand(raw)
	}
}

func (agent *Agent) Write(msg []byte) (int, error) {
	if atomic.LoadUint32(&agent.running) == 0 {
		return 0, fmt.Errorf("sending on stopped agent instance")
	}

	logger.Debug().Println(pkgName, "Sending: ", string(msg))
	return agent.controller.Write(msg)
}

// Loop is main loop of SyntropyStack agent
func (agent *Agent) Loop() {
	logger.Info().Println(pkgName, "Starting Agent messages handler")

	if !atomic.CompareAndSwapUint32(&agent.running, 0, 1) {
		logger.Warning().Println(pkgName, "Agent instance is already running")
		return
	}

	// Start all "services"
	agent.wgWatcher.Start()

	// Main agent loop - handles messages, received from controller
	go agent.messageHandler()
}

// Stop closes connections to controller and stops all runners
func (agent *Agent) Stop() {
	logger.Info().Println(pkgName, "Stopping Agent")
	if !atomic.CompareAndSwapUint32(&agent.running, 1, 0) {
		logger.Warning().Println(pkgName, "Agent instance is not running")

		return
	}

	// Stop all "services"
	agent.ping.Stop()
	agent.wgWatcher.Stop()

	// Close controler will also terminate agent loop
	agent.controller.Close()

	// cleanup
	agent.wg.Close()
	wireguard.DestroyAllInterfaces()
}
