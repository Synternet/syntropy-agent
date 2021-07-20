package config

import "time"

const version = "0.0.69"

type Location struct {
	Latitude  float32
	Longitude float32
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

	location      Location
	containerType string
}

var cache configCache
