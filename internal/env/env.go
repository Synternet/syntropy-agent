// Env packet describes all settings, common to whole application
package env

import "time"

const (
	// Controller is expecting ISO8601 time format
	// Technically ISO 8601 and RFC3339 are not technically the same thing.
	// But  RFC3339 is a stricter version of ISO8601. So it should be safe to use the RFC3339.
	TimeFormat = time.RFC3339
	// Prefix of all agent configured interfaces.
	InterfaceNamePrefix = "SYNTROPY_"
	// Public interface name suffix
	InterfaceNamePublicSuffix = "PUBLIC"
	// Default value for agent initiated messages to controller
	MessageDefaultID = "-"

	// Agent config directory
	SyntropyConfigDir = "/etc/syntropy"
	AgentConfigDir    = SyntropyConfigDir + "/platform"
	AgentConfigFile   = AgentConfigDir + "/config.yaml"
	AgentTempDir      = AgentConfigDir + "/tmp"

	// Locking agent to prevent several instances running
	LockFile = "/var/lock/syntropy"
)
