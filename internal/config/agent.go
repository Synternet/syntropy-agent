package config

import (
	"os"
	"strings"
)

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

func initAgentTags() {
	tags := strings.Split(os.Getenv("SYNTROPY_TAGS"), ",")
	for _, v := range tags {
		if len(v) > 0 {
			cache.agentTags = append(cache.agentTags, v)
		}
	}
}

func initControllerType() {
	switch strings.ToLower(os.Getenv("SYNTROPY_CONTROLLER_TYPE")) {
	case "saas":
		cache.controllerType = ControllerSaas
	case "script":
		cache.controllerType = ControllerScript
	case "blockchain":
		cache.controllerType = ControllerBlockchain
	default:
		cache.controllerType = ControllerSaas
	}
}
