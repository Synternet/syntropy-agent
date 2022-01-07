package config

import (
	"os"
	"strconv"
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
		if len(v) > 0 {
			cache.agentTags = append(cache.agentTags, v)
		}
	}
}

func initAgentToken() {
	cache.apiKey = os.Getenv("SYNTROPY_AGENT_TOKEN")
	if cache.apiKey == "" {
		cache.apiKey = os.Getenv("SYNTROPY_API_KEY")
	}
}

func initOwnerAddress() {
	cache.ownerAddress = os.Getenv("SYNTROPY_OWNER_ADDRESS")
}

func initIpfsURL() {
	cache.ipfsURL = "localhost:5001"
	url := os.Getenv("SYNTROPY_IPFS_URL")

	if url != "" {
		cache.ipfsURL = url
	}
}

func initCloudURL() {
	cache.cloudURL = "controller-prod-platform-agents.syntropystack.com"
	url := os.Getenv("SYNTROPY_CONTROLLER_URL")

	if url != "" {
		cache.cloudURL = url
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

func initCleanupOnExit() {
	cache.cleanupOnExit, _ = strconv.ParseBool(os.Getenv("SYNTROPY_CLEANUP_ON_EXIT"))
}

func initVPNClient() {
	cache.vpnClient, _ = strconv.ParseBool(os.Getenv("VPN_CLIENT"))
}
