package config

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
	ownerAddress   string // aka OWNER_ADDRESS
	ipfsURL        string
	controllerType int
	exporterPort   uint16

	agentName      string
	agentProvider  uint
	agentCategory  string
	servicesStatus bool
	agentTags      []string
	networkIDs     []string

	portsRange struct {
		start uint16
		end   uint16
	}
	mtu                 uint
	createIptablesRules bool

	debugLevel    int
	location      Location
	containerType string
	cleanupOnExit bool
	vpnClient     bool

	allowedIPs []AllowedIPEntry

	rerouteThresholds struct {
		diff  float32
		ratio float32
	}

	times struct {
		peerMonitor uint
	}
}

var cache configCache
