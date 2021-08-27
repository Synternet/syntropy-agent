package agent

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"github.com/SyntropyNet/syntropy-agent-go/agent/autoping"
	"github.com/SyntropyNet/syntropy-agent-go/agent/configinfo"
	"github.com/SyntropyNet/syntropy-agent-go/agent/dockerwatch"
	"github.com/SyntropyNet/syntropy-agent-go/agent/getinfo"
	"github.com/SyntropyNet/syntropy-agent-go/agent/peerdata"
	"github.com/SyntropyNet/syntropy-agent-go/agent/wgconf"
	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/SyntropyNet/syntropy-agent-go/controller/blockchain"
	"github.com/SyntropyNet/syntropy-agent-go/controller/saas"
	"github.com/SyntropyNet/syntropy-agent-go/controller/script"
	"github.com/SyntropyNet/syntropy-agent-go/docker"
	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/netfilter"
	"github.com/SyntropyNet/syntropy-agent-go/wireguard"
)

const pkgName = "SyntropyAgent. "

type Agent struct {
	running    uint32
	controller controller.Controller

	wg *wireguard.Wireguard

	commands map[string]controller.Command
	services []controller.Service
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

	//	agent.ping = pinger.NewPinger(agent)
	//	agent.wgWatcher = NewWgPeerWatcher(agent.wg, agent)

	agent.commands = make(map[string]controller.Command)
	agent.addCommand(getinfo.New(agent.controller))
	agent.addCommand(configinfo.New(agent.controller, agent.wg))
	agent.addCommand(wgconf.New(agent.controller, agent.wg))

	autoping := autoping.New(agent.controller)
	agent.addCommand(autoping)
	agent.addService(autoping)

	agent.addService(peerdata.New(agent.controller, agent.wg))

	if docker.IsDockerContainer() {
		agent.addService(dockerwatch.New(agent.controller))
	}

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
			logger.Info().Println(pkgName, "EOF. Closing message handler.")
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
	agent.startServices()

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
	agent.stopServices()

	// Close controler will also terminate agent loop
	agent.controller.Close()

	// cleanup
	agent.wg.Close()
	wireguard.DestroyAllInterfaces()
}
