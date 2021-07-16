package agent

import (
	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/SyntropyNet/syntropy-agent-go/controller/saas"
)

type Agent struct {
	controller controller.Controller
}

// NewAgent allocates instance of agent struct
// Parses shell environment and setups internal variables
func NewAgent(version string) (*Agent, error) {
	var err error
	agent := new(Agent)

	agent.controller, err = saas.NewCloudController(version)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

// Loop is main loop of SyntropyStack agent
func (agent *Agent) Loop() {
	go agent.controller.Start()
}

// Stop closes connections to controller and stops all runners
func (agent *Agent) Stop() {
	agent.controller.Stop()
}
