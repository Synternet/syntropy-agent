package agent

import (
	"context"
	"errors"
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
	"github.com/SyntropyNet/syntropy-agent-go/agent/supportinfo"
	"github.com/SyntropyNet/syntropy-agent-go/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent-go/agent/wgconf"
	"github.com/SyntropyNet/syntropy-agent-go/controller"
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
	ctx        context.Context
	controller common.Controller

	wg     *swireguard.Wireguard
	pm     *peermon.PeerMonitor
	router *router.Router

	commands map[string]common.Command
	services []common.Service
}

// NewAgent allocates instance of agent struct
// Parses shell environment and setups internal variables
func NewAgent(ctx context.Context, contype int) (*Agent, error) {
	var err error

	controller, err := controller.New(ctx, contype)
	if err != nil {
		return nil, err
	}

	// Config loggers early - to get more info logged
	logger.SetupGlobalLoger(controller, config.GetDebugLevel(), os.Stdout)

	agent := &Agent{
		ctx:        ctx,
		controller: controller,
		commands:   make(map[string]common.Command, 5),
		services:   make([]common.Service, 0, 5),
	}

	agent.pm = &peermon.PeerMonitor{}
	agent.router = router.New(ctx, agent.controller, agent.pm)
	agent.wg, err = swireguard.New(agent.pm)
	if err != nil {
		return nil, err
	}
	agent.wg.LogInfo()

	agent.addCommand(configinfo.New(agent.controller, agent.wg, agent.router))
	agent.addCommand(wgconf.New(agent.controller, agent.wg, agent.router))

	autoping := autoping.New(ctx, agent.controller)
	agent.addCommand(autoping)
	agent.addService(autoping)

	agent.addService(peerdata.New(ctx, agent.controller, agent.wg))
	agent.addService(agent.router)

	var dockerHelper docker.DockerHelper

	switch config.GetContainerType() {
	case config.ContainerTypeDocker:
		dockerWatch := docker.New(ctx, agent.controller)
		agent.addService(dockerWatch)
		dockerHelper = dockerWatch

	case config.ContainerTypeKubernetes:
		agent.addService(kubernetes.New(ctx, agent.controller))

	case config.ContainerTypeHost:
		agent.addService(hostnetsrv.New(ctx, agent.controller))

	default:
		logger.Warning().Println(pkgName, "unknown container type: ", config.GetContainerType())
	}

	if config.GetContainerType() != config.ContainerTypeDocker {
		dockerHelper = &docker.DockerNull{}
		netfilter.Disable()
	}

	agent.addCommand(getinfo.New(agent.controller, dockerHelper))
	agent.addCommand(supportinfo.New(agent.controller))

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
			logger.Info().Println(pkgName, "Controller EOF. Closing message handler.")
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

// Starts the agent and executes the message loop.
// Exits the loop after the context is closed.
// Also, cleans up everything.
func (agent *Agent) Run() error {
	err := agent.start()
	if err != nil {
		return err
	}
	defer agent.stop()

	for {
		select {
		case <-agent.ctx.Done():
			return nil
		default:
		}

		raw, err := agent.controller.Recv()

		if errors.Is(err, io.EOF) {
			// Stop runner if the reader is done
			logger.Info().Println(pkgName, "Controller EOF. Closing message handler.")
			return err
		} else if err != nil {
			// Simple errors are handled inside controller. This should be only fatal errors
			logger.Error().Println(pkgName, "Message handler error: ", err)
			return err
		}

		agent.processCommand(raw)
	}
}

func (agent *Agent) start() error {
	logger.Info().Println(pkgName, "Starting Agent messages handler")

	if agent.GetState() != stopped {
		logger.Warning().Println(pkgName, "Agent instance is already running")
		return nil
	}

	// Start all "services"
	return agent.startServices()
}

// Stop closes connections to controller and stops all runners
func (agent *Agent) stop() error {
	logger.Info().Println(pkgName, "Stopping Agent")
	if agent.GetState() == stopped {
		logger.Warning().Println(pkgName, "Agent instance is not running")

		return nil
	}

	// Stop all "services"
	err := agent.stopServices()
	if err != nil {
		logger.Warning().Println(pkgName, "Failed stopping services: ", err)
	}

	// Close controler will also terminate agent loop
	err = agent.controller.Close()
	if err != nil {
		logger.Warning().Println(pkgName, "Failed stopping the controller: ", err)
	}

	// Wireguard cleanup on exit
	return agent.wg.Close()
}
