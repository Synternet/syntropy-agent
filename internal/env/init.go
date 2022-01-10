package env

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func Init() {
	initAgentDirs()
}

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
		os.Exit(-int(unix.ENOTDIR))
	}

	// Cleanup previously cached private & public key files
	// We no longer rely on them
	// (maybe some day this code should also be removed?)
	cleanupObsoleteFiles(AgentConfigDir + "/privatekey-*")
	cleanupObsoleteFiles(AgentConfigDir + "/publickey-*")
	cleanupObsoleteFiles(AgentTempDir + "/config_dump")
}
