package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const maxPort = 65535

func Init() {
	var tmpval uint

	initString(&cache.apiKey, "SYNTROPY_AGENT_TOKEN", "")
	if cache.apiKey == "" {
		// Fallback. This was used on older agent versions
		initString(&cache.apiKey, "SYNTROPY_API_KEY", "")
	}
	initString(&cache.cloudURL, "SYNTROPY_CONTROLLER_URL",
		"controller-prod-platform-agents.syntropystack.com")

	initDeviceID()
	initControllerType()
	initDebugLevel()

	initString(&cache.ownerAddress, "SYNTROPY_OWNER_ADDRESS", "")
	initString(&cache.ipfsURL, "SYNTROPY_IPFS_URL", "localhost:5001")

	initUint(&tmpval, "SYNTROPY_EXPORTER_PORT", 0)
	if tmpval <= maxPort {
		cache.exporterPort = uint16(tmpval)
	}
	initUint(&cache.mtu, "SYNTROPY_MTU", 0)

	initAgentName()
	initUint(&cache.agentProvider, "SYNTROPY_PROVIDER", 0)
	cache.agentCategory = os.Getenv("SYNTROPY_CATEGORY")
	initBool(&cache.servicesStatus, "SYNTROPY_SERVICES_STATUS", false)
	initAgentTags()
	cache.networkIDs = strings.Split(os.Getenv("SYNTROPY_NETWORK_IDS"), ",")

	initPortsRange()
	initAllowedIPs()
	initIptables()

	initLocation()
	cache.containerType = strings.ToLower(os.Getenv("SYNTROPY_NETWORK_API"))
	initBool(&cache.cleanupOnExit, "SYNTROPY_CLEANUP_ON_EXIT", false)
	initBool(&cache.vpnClient, "VPN_CLIENT", false)

	// reroute thresholds used to compare better latency.
	// Default values: diff >= 10ms and at least 10% better
	cache.rerouteThresholds.diff = 10
	cache.rerouteThresholds.ratio = 1.1

	initUint(&cache.times.peerMonitor, "SYNTROPY_PEER_MONITOR_TIME", 5)
	if cache.times.peerMonitor < 1 {
		cache.times.peerMonitor = 1
	} else if cache.times.peerMonitor > 60 {
		cache.times.peerMonitor = 60
	}

}

func Close() {
	// Anything needed to be closed or destroyed at the end of program, goes here
}

func initLocation() {
	val, err := strconv.ParseFloat(os.Getenv("SYNTROPY_LAT"), 32)
	if err == nil {
		cache.location.Latitude = float32(val)
	}
	val, err = strconv.ParseFloat(os.Getenv("SYNTROPY_LON"), 32)
	if err == nil {
		cache.location.Longitude = float32(val)
	}
}

func initDebugLevel() {
	switch strings.ToUpper(os.Getenv("SYNTROPY_LOG_LEVEL")) {
	case "DEBUG":
		cache.debugLevel = logger.DebugLevel
	case "INFO":
		cache.debugLevel = logger.InfoLevel
	case "WARNING":
		cache.debugLevel = logger.WarningLevel
	case "ERROR":
		cache.debugLevel = logger.ErrorLevel
	default:
		cache.debugLevel = logger.InfoLevel
	}
}
