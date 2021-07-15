package agent

type Agent struc {
	url string
	token string
}

func NewAgent() (*Agent, error) {
	agent Agent {
		url: os.Getenv("SYNTROPY_CONTROLLER_URL"),
		token: os.Getenv("SYNTROPY_AGENT_TOKEN")
	}

	if agent.token == "" {
		return nil, "SYNTROPY_AGENT_TOKEN is not set"
	}

	if agent.url == "" {
		agent.url = "controller-prod-platform-agents.syntropystack.com"
	}

	return &agent, nil
}

func (agent *Agent) Run() {
	
}