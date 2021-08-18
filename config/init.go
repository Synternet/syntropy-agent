package config

import (
	"log"
	"os"
	"strings"
)

func Init() {
	initAgentDirs()

	initAgentToken()
	initCloudURL()
	initDeviceID()
	initControllerType()

	initAgentName()
	initAgentProvider()
	initAgentCategory()
	initServicesStatus()
	initAgentTags()
	initNetworkIDs()

	updatePublicIp()
	initPortsRange()

	initLocation()
	initContainer()

	log.Println("Config init completed")
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
	cache.location.Latitude = os.Getenv("SYNTROPY_LAT")
	cache.location.Longitude = os.Getenv("SYNTROPY_LON")
}
