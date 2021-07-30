package config

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

func Init() {
	initAgentDirs()

	initAgentToken()
	initCloudURL()
	initDeviceID()

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

func initAgentName() {
	var err error
	cache.agentName = os.Getenv("SYNTROPY_AGENT_NAME")

	if cache.agentName != "" {
		return
	}

	// Fallback to hostname, if shell variable `SYNTROPY_AGENT_NAME` is missing
	cache.agentName, err = os.Hostname()
	if err != nil {
		// Should hever happen, but its a good practice to handle all errors
		cache.agentName = "UnknownSyntropyAgent"
	}
}

func initAgentProvider() {
	str := os.Getenv("SYNTROPY_PROVIDER")
	val, err := strconv.Atoi(str)
	if err != nil {
		// SYNTROPY_PROVIDER is not set or is not an integer
		return
	}
	cache.agentProvider = val
}

func initAgentCategory() {
	cache.agentCategory = os.Getenv("SYNTROPY_CATEGORY")
}

func initServicesStatus() {
	cache.servicesStatus = false
	str := os.Getenv("SYNTROPY_SERVICES_STATUS")
	if strings.ToLower(str) == "true" {
		cache.servicesStatus = true
	}
}

func initAgentTags() {
	tags := strings.Split(os.Getenv("SYNTROPY_TAGS"), ",")
	for _, v := range tags {
		if len(v) > 3 {
			cache.agentTags = append(cache.agentTags, v)
		}
	}
}

func initDeviceID() {
	productUUID := func() (string, error) {
		data, err := ioutil.ReadFile("/sys/class/dmi/id/product_uuid")
		if err != nil {
			return "", err
		}
		return strings.Trim(string(data), "\n"), nil
	}

	machineID := func() (string, error) {
		data, err := ioutil.ReadFile("/etc/machine-id")
		if err != nil {
			return "", err
		}
		return strings.Trim(string(data), "\n") + GetPublicIp(), nil
	}

	cpuSerial := func() string {
		var serial string

		// This works on Raspberry PI linux.
		// But is not working on generic PC linux
		file, err := os.Open("/proc/cpuinfo")
		if err != nil {
			return "0000000000000000" + GetPublicIp()
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			sarr := strings.Split(scanner.Text(), ":")
			if len(sarr) >= 2 && strings.Contains(sarr[0], "Serial") {
				serial = strings.TrimSpace(sarr[1])
				break
			}
		}

		// Fallback to any sane value
		if serial == "" {
			serial = "0000000000000000" + GetPublicIp()
		}

		return serial
	}

	devID, err := productUUID()
	if err != nil {
		devID, err = machineID()
	}
	if err != nil {
		devID = cpuSerial()
	}

	cache.deviceID = devID
}

func initAgentToken() {
	cache.apiKey = os.Getenv("SYNTROPY_AGENT_TOKEN")

	if cache.apiKey == "" {
		log.Fatal("SYNTROPY_AGENT_TOKEN is not set")
	}
}

func initCloudURL() {
	cache.cloudURL = "controller-prod-platform-agents.syntropystack.com"
	url := os.Getenv("SYNTROPY_CONTROLLER_URL")

	// TODO maybe add try DNS resove here ?
	if url != "" {
		cache.cloudURL = url
	}
}

func initLocation() {
	cache.location.Latitude = os.Getenv("SYNTROPY_LAT")
	cache.location.Longitude = os.Getenv("SYNTROPY_LON")
}
