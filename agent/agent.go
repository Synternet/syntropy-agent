package agent

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/SyntropyNet/syntropy-agent/agent/autoping"
	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/configinfo"
	"github.com/SyntropyNet/syntropy-agent/agent/docker"
	"github.com/SyntropyNet/syntropy-agent/agent/exporter"
	"github.com/SyntropyNet/syntropy-agent/agent/getinfo"
	"github.com/SyntropyNet/syntropy-agent/agent/hostnetsrv"
	"github.com/SyntropyNet/syntropy-agent/agent/ifacemon"
	"github.com/SyntropyNet/syntropy-agent/agent/kubernetes"
	"github.com/SyntropyNet/syntropy-agent/agent/mole"
	"github.com/SyntropyNet/syntropy-agent/agent/peerwatch"
	"github.com/SyntropyNet/syntropy-agent/agent/settings"
	"github.com/SyntropyNet/syntropy-agent/agent/supportinfo"
	"github.com/SyntropyNet/syntropy-agent/agent/supportinfo/shellcmd"
	"github.com/SyntropyNet/syntropy-agent/agent/wgconf"
	"github.com/SyntropyNet/syntropy-agent/controller"
	"github.com/SyntropyNet/syntropy-agent/controller/blockchain"
	"github.com/SyntropyNet/syntropy-agent/controller/saas"
	"github.com/SyntropyNet/syntropy-agent/controller/script"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
	"github.com/SyntropyNet/syntropy-agent/pkg/pubip"
)

const pkgName = "SyntropyAgent. "

type Agent struct {
	// main controller agent is communicating to.
	// Is used as io.Writer to send messages and Recv to read messages from
	controller controller.Controller

	// context is for all agent's childs, like "services"
	// Agent itself is not dependent on this context
	ctx    context.Context
	cancel context.CancelFunc

	// various helpers, used crossed-services
	pinger *multiping.MultiPing
	mole   *mole.Mole

	// services and commands slice/map
	commands map[string]common.Command
	services []common.Service
}

// New allocates instance of agent struct
// Parses shell environment and setups internal variables
func New(contype int) (*Agent, error) {
	var err error
	var controller controller.Controller

	switch contype {
	case config.ControllerSaas:
		controller, err = saas.New()
		if controller != nil {
		}
	case config.ControllerScript:
		controller, err = script.New()
	case config.ControllerBlockchain:
		controller, err = blockchain.New()
	default:
		err = fmt.Errorf("unexpected controller type %d", contype)
	}
	if err != nil {
		logger.Error().Println(pkgName, "Controller init failure", err)
		return nil, err
	}
	logger.Info().Println(pkgName, "Public IP: ", pubip.GetPublicIp().String(), "/", pubip.Provider())

	agent := &Agent{
		controller: controller,
		commands:   make(map[string]common.Command),
		services:   make([]common.Service, 0),
	}
	agent.ctx, agent.cancel = context.WithCancel(context.Background())

	agent.mole, err = mole.New(agent.controller)
	if err != nil {
		logger.Error().Println(pkgName, "Mole init failure", err)
		return nil, err
	}

	agent.pinger, err = multiping.New(true)
	if err != nil {
		logger.Error().Println(pkgName, "Multiping init failure", err)
		return nil, err
	}

	agent.mole.Wireguard().LogInfo()

	var dockerHelper docker.DockerHelper

	switch config.GetContainerType() {
	case config.ContainerTypeDocker:
		dockerWatch := docker.New(agent.controller)
		agent.addService(dockerWatch)
		dockerHelper = dockerWatch
		// SYNTROPY_CHAIN iptables rule is created only in Docker case
		err = agent.mole.CreateChain()
		if err != nil {
			logger.Error().Println(pkgName, "Syntropy chain create:", err)
		}

	case config.ContainerTypeKubernetes:
		agent.addService(kubernetes.New(agent.controller))

	case config.ContainerTypeHost:
		agent.addService(hostnetsrv.New(agent.controller))

	default:
		logger.Warning().Println(pkgName, "unknown SYNTROPY_NETWORK_API type: ", config.GetContainerType())
	}

	if config.GetContainerType() != config.ContainerTypeDocker {
		dockerHelper = &docker.DockerNull{}
	}

	agent.addCommand(configinfo.New(agent.controller, agent.mole, dockerHelper))
	agent.addCommand(wgconf.New(agent.controller, agent.mole))

	autoping := autoping.New(agent.controller, agent.pinger)
	agent.addCommand(autoping)
	agent.addService(autoping)
	agent.addService(peerwatch.New(agent.controller, agent.mole, agent.pinger))
	// For SaaS Controller add public IP change monitor with reconnect callback
	switch c := controller.(type) {
	case *saas.CloudController:
		agent.addService(ifacemon.New(c.Reconnect))
	}

	if config.MetricsExporterEnabled() {
		metrics, err := exporter.New(config.MetricsExporterPort(), agent.mole.Router())
		if err != nil {
			logger.Error().Println(pkgName, "metrics exporter create", err)
		} else {
			agent.addService(metrics)
		}
	}

	agent.addCommand(getinfo.New(agent.controller, dockerHelper))
	agent.addCommand(settings.New())
	agent.addCommand(supportinfo.New(agent.controller,
		shellcmd.New("wg_info", "wg", "show"),
		shellcmd.New("routes", "route", "-n"),
		autoping))

	return agent, agent.controller.Open()
}

// Starts worker services and executes the message loop.
// This loop is terminated by Close()
func (agent *Agent) Run() {
	logger.Info().Println(pkgName, "Starting Agent messages handler")
	// Start all "services"
	agent.startServices()

	// Decouple packet receive from processing it
	// Because a big network configuration apply takes quite a long time
	// And this can result in websocket timeout
	// Use buffered channel and 2 goroutines for this purpose
	rxchan := make(chan []byte, 9)

	go func() {
		for raw := range rxchan {
			agent.processCommand(raw)
		}
	}()

	go func() {
		defer close(rxchan)

		for {
			raw, err := agent.controller.Recv()

			if errors.Is(err, io.EOF) {
				// Stop runner if the reader is done
				logger.Info().Println(pkgName, "Controller EOF. Closing message handler.")
				return
			} else if err != nil {
				// Simple errors are handled inside controller.
				// This should be only fatal non recovery errors
				// Actually this should never happen.
				logger.Error().Println(pkgName, "Message handler error: ", err)
				return
			}

			rxchan <- raw
		}
	}()
}

// Close closes connections to controller and stops all runners
// P.S. The naming dilemma: Stop vs Close
// And I choose Close, because I can use Closer interface
func (agent *Agent) Close() error {
	logger.Info().Println(pkgName, "Stopping Agent")

	// Stop all "services"
	agent.stopServices()

	// cleanup on exit (craftman mole knows what to cleanup)
	logger.Debug().Println(pkgName, "Mole close")
	err := agent.mole.Close()
	if err != nil {
		logger.Error().Println(pkgName, "mole cleanup:", err)
	}

	// Close controler will also terminate agent loop
	logger.Debug().Println(pkgName, "Closing connection with controller")
	err = agent.controller.Close()
	if err != nil {
		logger.Warning().Println(pkgName, "Failed stopping the controller: ", err)
	}
	logger.Debug().Println(pkgName, "Agent close done")

	return nil
}

// Writer returns pointer to controller, as a io.Writer interface
func (agent *Agent) Writer() io.Writer {
	return agent.controller
}
