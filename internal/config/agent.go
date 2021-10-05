package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
)

const (
	AgentConfigDir  = "/etc/syntropy-agent"
	AgentConfigFile = AgentConfigDir + "/config.yaml"
	AgentTempDir    = AgentConfigDir + "/tmp"
)

func cleanupObsoleteFiles(patern string) {
	fileNames, err := filepath.Glob(patern)
	if err == nil {
		for _, f := range fileNames {
			os.Remove(f)
		}
	}
}

func initAgentDirs() {
	// MkdirAll is equivalent of mkdir -p, so it will not recreate existing dirs
	// And I can simplify my code and do not check if dirs already exist
	err := os.MkdirAll(AgentConfigDir, 0700)
	if err != nil {
		logger.Error().Printf("%s Config dir %s: %s\n", pkgName, AgentConfigDir, err.Error())
		os.Exit(-2)
	}
	err = os.MkdirAll(AgentTempDir, 0700)
	if err != nil {
		logger.Error().Printf("%s Temp dir %s: %s\n", pkgName, AgentTempDir, err.Error())
		os.Exit(-2)
	}

	// Cleanup previously cached private & public key files
	// We no longer rely on them
	// (maybe some day this code should also be removed?)
	cleanupObsoleteFiles(AgentConfigDir + "/privatekey-*")
	cleanupObsoleteFiles(AgentConfigDir + "/publickey-*")
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
		logger.Error().Println(pkgName, "SYNTROPY_AGENT_TOKEN is not set")
		os.Exit(-1)
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
