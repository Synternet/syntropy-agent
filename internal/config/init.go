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
	initControllerType()
	initString(&cache.cloudURL, "SYNTROPY_CONTROLLER_URL",
		"controller-prod-platform-agents.syntropystack.com")

	initString(&cache.ipfsURL, "SYNTROPY_IPFS_URL", "localhost:5001")
	initString(&cache.ownerAddress, "SYNTROPY_OWNER_ADDRESS", "")

	initDebugLevel()

	initUint(&cache.mtu, "SYNTROPY_MTU", 0)
	initAgentName()
	initUint(&cache.agentProvider, "SYNTROPY_PROVIDER", 0)
	initAgentTags()
	initBool(&cache.servicesStatus, "SYNTROPY_SERVICES_STATUS", false)
	initPortsRange()
	cache.containerType = strings.ToLower(os.Getenv("SYNTROPY_NETWORK_API"))

	var k8sNamespaces string
	initString(&k8sNamespaces, "SYNTROPY_NAMESPACE", "")
	cache.kubernetesNamespaces = strings.Split(k8sNamespaces, ",")

	initAllowedIPs()
	initLocation()
	initBool(&cache.vpnClient, "VPN_CLIENT", false)
	initIptables()
	initBool(&cache.cleanupOnExit, "SYNTROPY_CLEANUP_ON_EXIT", false)

	initUint(&tmpval, "SYNTROPY_EXPORTER_PORT", 0)
	if tmpval <= maxPort {
		cache.exporterPort = uint16(tmpval)
	}

	initUint(&cache.times.peerMonitor, "SYNTROPY_PEERCHECK_TIME", 5)
	if cache.times.peerMonitor < 1 {
		cache.times.peerMonitor = 1
	} else if cache.times.peerMonitor > 60 {
		cache.times.peerMonitor = 60
	}
	initUint(&cache.times.rerouteWindow, "SYNTROPY_PEERCHECK_WINDOW", 24)
	if cache.times.rerouteWindow < 1 {
		cache.times.rerouteWindow = 1
	}
	initUint(&cache.routeDelThreshold, "SYNTROPY_ROUTEDEL_THRESHOLD", 0)

	initUint(&cache.times.websocketTimeout, "SYNTROPY_WSS_TIMEOUT", 0)

	initDeviceID()

	// reroute thresholds used to compare better latency.
	// Default values: diff >= 10ms and at least 10% better
	cache.rerouteThresholds.diff = 10
	cache.rerouteThresholds.ratio = 1.1
	initRouteStrategy()
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
	case "MESSAGE", "MSG":
		cache.debugLevel = logger.MessageLevel
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

func initRouteStrategy() {
	switch strings.ToUpper(os.Getenv("SYNTROPY_ROUTE_STRATEGY")) {
	case "speed":
		cache.routeStrategy = RouteStrategySpeed
	case "other":
		cache.routeStrategy = RouteStrategyDirectRoute
	default:
		cache.routeStrategy = RouteStrategySpeed
	}
}
