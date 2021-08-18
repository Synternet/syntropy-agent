package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	AgentConfigDir  = "/etc/syntropy-agent"
	AgentConfigFile = AgentConfigDir + "/config.yaml"
	AgentTempDir    = AgentConfigDir + "/tmp"
)

func initAgentDirs() {
	// MkdirAll is equivalent of mkdir -p, so it will not recreate existing dirs
	// And I can simplify my code and do not check if dirs already exist
	err := os.MkdirAll(AgentConfigDir, 0700)
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(AgentTempDir, 0700)
	if err != nil {
		log.Fatal(err)
	}

}

func initAgentName() {
	var err error
	cache.agentName = os.Getenv("SYNTROPY_AGENT_NAME")

	if cache.agentName != "" {
		return
	}

	// Fallback to hostname, if shell variable `SYNTROPY_AGENT_NAME` is missing
	cache.agentName, err = os.Hostname()
	if err != nil {
		// Should hever happen, but its a good practice to handle all errors
		cache.agentName = "UnknownSyntropyAgent"
	}
}

func initAgentProvider() {
	str := os.Getenv("SYNTROPY_PROVIDER")
	val, err := strconv.Atoi(str)
	if err != nil {
		// SYNTROPY_PROVIDER is not set or is not an integer
		return
	}
	cache.agentProvider = val
}

func initAgentCategory() {
	cache.agentCategory = os.Getenv("SYNTROPY_CATEGORY")
}

func initAgentTags() {
	tags := strings.Split(os.Getenv("SYNTROPY_TAGS"), ",")
	for _, v := range tags {
		if len(v) > 3 {
			cache.agentTags = append(cache.agentTags, v)
		}
	}
}

func initAgentToken() {
	cache.apiKey = os.Getenv("SYNTROPY_AGENT_TOKEN")

	if cache.apiKey == "" {
		log.Fatal("SYNTROPY_AGENT_TOKEN is not set")
	}
}

func initCloudURL() {
	cache.cloudURL = "controller-prod-platform-agents.syntropystack.com"
	url := os.Getenv("SYNTROPY_CONTROLLER_URL")

	// TODO maybe add try DNS resove here ?
	if url != "" {
		cache.cloudURL = url
	}
}

func initControllerType() {
	// Always default to Software-as-a-Service
	cache.controllerType = ControllerSaas

	sct := os.Getenv("SYNTROPY_CONTROLLER_TYPE")
	switch strings.ToLower(sct) {
	case "saas":
		cache.controllerType = ControllerSaas
	case "script":
		cache.controllerType = ControllerScript
	case "blockchain":
		cache.controllerType = ControllerBlockchain
	case "":
		// If env variable is unset - stick with default
	default:
		log.Printf("Unknown controller type `%s`. Using default controller\n", sct)
	}
}
