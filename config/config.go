package config

import "time"

const version = "0.0.69"
const subversion = "local"

type Location struct {
	Latitude  string
	Longitude string
}

// This struct is used to cache commonly used Syntropy agent configuration
// some of them are exported shell variables, some are parsed from OS settings
// Some may be generated.
// Cache them and use from here
type configCache struct {
	apiKey   string // aka AGENT_TOKEN
	cloudURL string
	deviceID string

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

	location      Location
	containerType string
	dockerNetInfo []DockerNetworkInfoEntry
}

var cache configCache
