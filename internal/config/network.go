package config

import (
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const ipUpdatePeriod = 5 * time.Minute

func updatePublicIp() {
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

	cache.publicIP.ip = ip
	cache.publicIP.updated = time.Now()
}

func initNetworkIDs() {
	cache.networkIDs = strings.Split(os.Getenv("SYNTROPY_NETWORK_IDS"), ",")
}

func initPortsRange() {
	const maxPort = 65535
	// Init to sane defaults
	cache.portsRange.start = 49152
	cache.portsRange.end = maxPort

	strport := strings.Split(os.Getenv("SYNTROPY_PORT_RANGE"), "-")
	if len(strport) != 2 {
		return
	}
	p1, e1 := strconv.Atoi(strport[0])
	p2, e2 := strconv.Atoi(strport[0])
	if e1 != nil || e2 != nil ||
		p1 <= 0 || p2 <= 0 ||
		p1 > maxPort || p2 > maxPort {
		return
	}

	cache.portsRange.start = uint16(p1)
	cache.portsRange.end = uint16(p2)
}