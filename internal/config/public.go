package config

import "time"

const (
	ControllerSaas = iota
	ControllerScript
	ControllerBlockchain
	ControllerUnknown
)

const (
	ContainerTypeDocker     = "docker"
	ContainerTypeKubernetes = "kubernetes"
	ContainerTypeHost       = "host"
)

func GetControllerType() int {
	return cache.controllerType
}

func GetControllerName(ctype int) string {
	switch ctype {
	case ControllerSaas:
		return "SaaS (cloud)"
	case ControllerScript:
		return "Script"
	case ControllerBlockchain:
		return "Blockchain"
	default:
		return "Unknown"
	}
}

func GetDebugLevel() int {
	return cache.debugLevel
}

func GetAgentToken() string {
	return cache.apiKey
}

func GetCloudURL() string {
	return cache.cloudURL
}

func GetOwnerAddress() string {
	return cache.ownerAddress
}

func GetIpfsUrl() string {
	return cache.ipfsURL
}

func GetAgentName() string {
	return cache.agentName
}

func GetAgentProvider() uint {
	return cache.agentProvider
}

func GetServicesStatus() bool {
	return cache.servicesStatus
}

func GetAgentTags() []string {
	if len(cache.agentTags) > 0 {
		return cache.agentTags
	} else {
		return []string{}
	}
}

func GetPortsRange() (uint16, uint16) {
	return cache.portsRange.start, cache.portsRange.end
}

func GetInterfaceMTU() uint {
	return cache.mtu
}

func CreateIptablesRules() bool {
	return cache.createIptablesRules
}

func GetDeviceID() string {
	return cache.deviceID
}

func GetContainerType() string {
	return cache.containerType
}

func GetNamespace() string {
	return cache.kubernetesNamespace
}

func GetLocationLatitude() float32 {
	return cache.location.Latitude
}

func GetLocationLongitude() float32 {
	return cache.location.Longitude
}

func CleanupOnExit() bool {
	return cache.cleanupOnExit
}

func GetHostAllowedIPs() []AllowedIPEntry {
	return cache.allowedIPs
}

func IsVPNClient() bool {
	return cache.vpnClient
}

func SetRerouteThresholds(diff, ratio float32) {
	cache.rerouteThresholds.diff = diff
	cache.rerouteThresholds.ratio = ratio
}

func RerouteThresholds() (float32, float32) {
	return cache.rerouteThresholds.diff, cache.rerouteThresholds.ratio
}

func MetricsExporterEnabled() bool {
	return cache.exporterPort > 0
}

func MetricsExporterPort() uint16 {
	return cache.exporterPort
}

func PeerCheckTime() time.Duration {
	return time.Second * time.Duration(cache.times.peerMonitor)
}

func PeerCheckWindow() uint {
	return cache.times.rerouteWindow
}

func GetRouteDeleteThreshold() uint {
	return cache.routeDelThreshold
}
