package config

import "time"

func GetVersion() string {
	return version
}

func GetAgentToken() string {
	return cache.apiKey
}

func GetCloudURL() string {
	return cache.cloudURL
}

func GetAgentName() string {
	return cache.agentName
}

func GetAgentProvider() int {
	return cache.agentProvider
}

func GetAgentCategory() string {
	return cache.agentCategory
}

func GetServicesStatus() bool {
	return cache.servicesStatus
}

func GetAgentTags() []string {
	return cache.agentTags
}

func GetPublicIp() string {
	if time.Now().After(cache.publicIP.updated.Add(ipUpdatePeriod)) {
		updatePublicIp()
	}

	return cache.publicIP.ip
}

func GetDeviceID() string {
	return cache.deviceID
}
