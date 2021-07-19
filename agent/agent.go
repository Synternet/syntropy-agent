package agent

import (
	"log"

	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/SyntropyNet/syntropy-agent-go/controller/saas"
)

type Agent struct {
	controller controller.Controller
	msgChan    chan []byte

	commands map[string]func(a *Agent, req []byte) ([]byte, error)
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
	agent.msgChan = make(chan []byte)

	agent.commands = make(map[string]func(a *Agent, req []byte) ([]byte, error))
	agent.commands["AUTO_PING"] = autoPing
	agent.commands["GET_INFO"] = getInfo

	return agent, nil
}

func (agent *Agent) messageHadler() {
	var err error
	for {
		raw, ok := <-agent.msgChan
		// Stop runner if the channel is closed
		if !ok {
			return
		}

		err = agent.processCommand(raw)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

// Loop is main loop of SyntropyStack agent
func (agent *Agent) Loop() {
	go agent.messageHadler()
	go agent.controller.Start(agent.msgChan)
}

// Stop closes connections to controller and stops all runners
func (agent *Agent) Stop() {
	agent.controller.Stop()
	close(agent.msgChan)
}
