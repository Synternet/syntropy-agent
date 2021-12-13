package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

func Init() {
	initAgentDirs()

	initAgentToken()
	initCloudURL()
	initDeviceID()
	initControllerType()
	initDebugLevel()

	initAgentName()
	initAgentProvider()
	initAgentCategory()
	initServicesStatus()
	initAgentTags()
	initNetworkIDs()

	initPortsRange()
	initAllowedIPs()
	initMTU()
	initIptables()

	initLocation()
	initContainer()
	initCleanupOnExit()
	initVPNClient()

	initThresholds()
}

func Close() {
	// Anything needed to be closed or destroyed at the end of program, goes here
}

func initServicesStatus() {
	cache.servicesStatus = false
	str := os.Getenv("SYNTROPY_SERVICES_STATUS")
	if strings.ToLower(str) == "true" {
		cache.servicesStatus = true
	}
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

func initContainer() {
	cache.containerType = strings.ToLower(os.Getenv("SYNTROPY_NETWORK_API"))
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

func initThresholds() {
	// reroute thresholds used to compare better latency.
	// Default values: diff >= 10ms and at least 10% better
	cache.rerouteThresholds.diff = 10
	cache.rerouteThresholds.ratio = 1.1
}
