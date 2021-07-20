package config

import "time"

const version = "0.0.69"

// This struct is used to cache commonly used Syntropy agent configuration
// some of them are exported shell variables, some are parsed from OS settings
// Some may be generated.
// Cache them and use from here
type configCache struct {
	apiKey   string // aka AGENT_TOKEN
	cloudURL string

	agentName      string
	agentProvider  int
	agentCategory  string
	servicesStatus bool
	agentTags      []string
	deviceID       string

	publicIP struct {
		ip      string
		updated time.Time
	}
}

var cache configCache
