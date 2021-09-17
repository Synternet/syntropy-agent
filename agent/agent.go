package agent

import (
	"fmt"
	"io"
	"os"

	"github.com/SyntropyNet/syntropy-agent-go/agent/autoping"
	"github.com/SyntropyNet/syntropy-agent-go/agent/configinfo"
	"github.com/SyntropyNet/syntropy-agent-go/agent/docker"
	"github.com/SyntropyNet/syntropy-agent-go/agent/getinfo"
	"github.com/SyntropyNet/syntropy-agent-go/agent/hostnetsrv"
	"github.com/SyntropyNet/syntropy-agent-go/agent/kubernetes"
	"github.com/SyntropyNet/syntropy-agent-go/agent/peerdata"
	"github.com/SyntropyNet/syntropy-agent-go/agent/router"
	"github.com/SyntropyNet/syntropy-agent-go/agent/wgconf"
	"github.com/SyntropyNet/syntropy-agent-go/agent/wireguard"
	"github.com/SyntropyNet/syntropy-agent-go/controller/blockchain"
	"github.com/SyntropyNet/syntropy-agent-go/controller/saas"
	"github.com/SyntropyNet/syntropy-agent-go/controller/script"
	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/internal/peermon"
	"github.com/SyntropyNet/syntropy-agent-go/netfilter"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/state"
)

const pkgName = "SyntropyAgent. "
const (
	stopped = iota
	running
)

type Agent struct {
	state.StateMachine
	controller common.Controller

	wg     *wireguard.Wireguard
	pm     *peermon.PeerMonitor
	router *router.Router

	commands map[string]common.Command
	services []common.Service
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

	agent.pm = &peermon.PeerMonitor{}
	agent.router = router.New(agent.controller, agent.pm)
	agent.wg, err = wireguard.New(agent.router, agent.pm)
	if err != nil {
		return nil, err
	}

	agent.commands = make(map[string]common.Command)
	agent.addCommand(configinfo.New(agent.controller, agent.wg))
	agent.addCommand(wgconf.New(agent.controller, agent.wg))

	autoping := autoping.New(agent.controller)
	agent.addCommand(autoping)
	agent.addService(autoping)

	agent.addService(peerdata.New(agent.controller, agent.wg))
	agent.addService(agent.router)

	var dockerHelper docker.DockerHelper

	switch config.GetContainerType() {
	case config.ContainerTypeDocker:
		dockerWatch := docker.New(agent.controller)
		agent.addService(dockerWatch)
		dockerHelper = dockerWatch

	case config.ContainerTypeKubernetes:
		agent.addService(kubernetes.New(agent.controller))

	case config.ContainerTypeHost:
		agent.addService(hostnetsrv.New(agent.controller))

	default:
		logger.Warning().Println(pkgName, "unknown container type: ", config.GetContainerType())
	}

	if config.GetContainerType() != config.ContainerTypeDocker {
		dockerHelper = &docker.DockerNull{}
		netfilter.Disable()
	}

	agent.addCommand(getinfo.New(agent.controller, dockerHelper))

	netfilter.CreateChain()

	return agent, nil
}

func (agent *Agent) messageHandler() {
	// Change state on start
	if !agent.ChangeState(stopped, running) {
		logger.Warning().Println(pkgName, "could not start. Already started ?")
		return
	}
	// Mark as not running on exit
	defer agent.SetState(stopped)

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
	if agent.GetState() != running {
		return 0, fmt.Errorf("sending on stopped agent instance")
	}

	return agent.controller.Write(msg)
}

// Loop is main loop of SyntropyStack agent
func (agent *Agent) Loop() {
	logger.Info().Println(pkgName, "Starting Agent messages handler")

	if agent.GetState() != stopped {
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
	if agent.GetState() == stopped {
		logger.Warning().Println(pkgName, "Agent instance is not running")

		return
	}

	// Stop all "services"
	agent.stopServices()

	// Close controler will also terminate agent loop
	agent.controller.Close()

	// cleanup
	agent.wg.Close()
	// TODO: add configuration
	// Usualy wireguard interfaces should not be destroyed
	// (e.g. app crash or agent upgrades should keep the network working)
	// But is a good practice to cleanup after yourself.
	// Also makes devel&debug stage easier
	wireguard.DestroyAllInterfaces()
}
