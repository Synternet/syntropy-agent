package config

import (
	"log"
	"os"
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
