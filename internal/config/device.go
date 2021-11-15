package config

import (
	"bufio"
	"io/ioutil"
	"os"
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/pkg/pubip"
)

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
		return strings.Trim(string(data), "\n") + pubip.GetPublicIp().String(), nil
	}

	cpuSerial := func() string {
		var serial string

		// This works on Raspberry PI linux.
		// But is not working on generic PC linux
		file, err := os.Open("/proc/cpuinfo")
		if err != nil {
			return "0000000000000000" + pubip.GetPublicIp().String()
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
			serial = "0000000000000000" + pubip.GetPublicIp().String()
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
