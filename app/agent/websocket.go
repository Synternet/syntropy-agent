package agent

import (
	"bufio"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"golang.org/x/net/websocket"
)

func (agent *Agent) CreateWebsocketConnection() error {
	url := url.URL{Scheme: "wss", Host: agent.url}
	wsConfig, err := websocket.NewConfig(url.String(), "http://localhost")
	if err != nil {
		return err
	}

	// Without these headers connection will be ignored silently
	wsConfig.Header.Set("authorization", agent.token)
	wsConfig.Header.Set("x-deviceid", generateDeviceId())
	wsConfig.Header.Set("x-deviceip", getPublicIp())
	wsConfig.Header.Set("x-devicename", getAgentName())
	wsConfig.Header.Set("x-devicestatus", "OK")
	wsConfig.Header.Set("x-agenttype", "Linux")
	wsConfig.Header.Set("x-agentversion", agent.version)

	agent.ws, err = websocket.DialConfig(wsConfig)
	if err != nil {
		return err
	}

	return nil
}

func generateDeviceId() string {
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
		return strings.Trim(string(data), "\n") + getPublicIp(), nil
	}

	cpuSerial := func() string {
		var serial string

		// This works on Raspberry PI linux.
		// But is not working on generic PC linux
		file, err := os.Open("/proc/cpuinfo")
		if err != nil {
			return "0000000000000000" + getPublicIp()
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
			serial = "0000000000000000" + getPublicIp()
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

	return devID
}

func getPublicIp() string {
	ip := "127.0.0.1" // sane fallback default

	ipProviders := []string{"https://ip.syntropystack.com",
		"https://ident.me",
		"https://ifconfig.me/ip",
		"https://ifconfig.co/ip"}

	for _, url := range ipProviders {
		resp, err := http.Get(url)
		if err != nil {
			// This provider failed, continue to next
			continue
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			// Could not parse body. Should not happen. Continue to next
			continue
		}

		// Some providers return IP address escaped in commas. Trim the newline as well,
		ip = strings.Trim(strings.Trim(string(body), "\n"), "\"")
		break
	}

	return ip
}

func getAgentName() string {
	name := os.Getenv("SYNTROPY_AGENT_NAME")

	if name != "" {
		return name
	}

	// Fallback to hostname, if shell variable `SYNTROPY_AGENT_NAME` is missing
	name, err := os.Hostname()
	if err != nil {
		// Should hever happen, but its a good practice to handle all errors
		name = "UnknownSyntropyAgent"
	}

	return name
}
