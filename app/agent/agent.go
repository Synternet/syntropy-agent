package agent

import (
	"fmt"
	"os"

	"github.com/gorilla/websocket"
)

type Agent struct {
	url     string
	token   string
	version string

	quit chan int

	ws *websocket.Conn
}

// NewAgent allocates instance of agent struct
// Parses shell environment and setups internal variables
func NewAgent(version string) (*Agent, error) {
	agent := Agent{
		url:     os.Getenv("SYNTROPY_CONTROLLER_URL"),
		token:   os.Getenv("SYNTROPY_AGENT_TOKEN"),
		version: version,
	}

	if agent.token == "" {
		return nil, fmt.Errorf("SYNTROPY_AGENT_TOKEN is not set")
	}

	if agent.url == "" {
		agent.url = "controller-prod-platform-agents.syntropystack.com"
	}

	err := agent.CreateWebsocketConnection()
	if err != nil {
		return nil, err
	}

	agent.quit = make(chan int)

	return &agent, nil
}

// Run is main loop of SyntropyStack agent
func (agent *Agent) Run() {

}

// Close closes websocket connection
func (agent *Agent) Close() {
	close(agent.quit)
	agent.ws.Close()
}
