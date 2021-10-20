package config

import "time"

const pkgName = "SyntropyAgentConfig. "

type Location struct {
	Latitude  float32
	Longitude float32
}

type AllowedIPEntry struct {
	Name   string
	Subnet string
}

// This struct is used to cache commonly used Syntropy agent configuration
// some of them are exported shell variables, some are parsed from OS settings
// Some may be generated.
// Cache them and use from here
type configCache struct {
	apiKey         string // aka AGENT_TOKEN
	cloudURL       string
	deviceID       string
	controllerType int

	agentName      string
	agentProvider  int
	agentCategory  string
	servicesStatus bool
	agentTags      []string
	networkIDs     []string

	publicIP struct {
		ip      string
		updated time.Time
	}
	portsRange struct {
		start uint16
		end   uint16
	}

	debugLevel    int
	location      Location
	containerType string
	cleanupOnExit bool

	allowedIPs []AllowedIPEntry
}

var cache configCache
