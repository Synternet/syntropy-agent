package config

import (
	"encoding/json"
	"io"
	"net"
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

	cache.portsRange.start = 0
	cache.portsRange.end = 0

	strport := strings.Split(os.Getenv("SYNTROPY_PORT_RANGE"), "-")
	if len(strport) != 2 {
		return
	}
	p1, e1 := strconv.Atoi(strport[0])
	p2, e2 := strconv.Atoi(strport[1])
	if e1 != nil || e2 != nil ||
		p1 <= 0 || p2 <= 0 ||
		p1 > maxPort || p2 > maxPort {
		return
	}

	// expect users to set range correctly, but still validate
	if p2 > p1 {
		cache.portsRange.start = uint16(p1)
		cache.portsRange.end = uint16(p2)
	} else {
		cache.portsRange.start = uint16(p2)
		cache.portsRange.end = uint16(p1)
	}
}

func initAllowedIPs() {
	cache.allowedIPs = []AllowedIPEntry{}
	str := os.Getenv("SYNTROPY_ALLOWED_IPS")

	var objMap []map[string]string
	err := json.Unmarshal([]byte(str), &objMap)
	if err != nil {
		return
	}

	for _, pair := range objMap {
		for k, v := range pair {
			// A very simple CIDR validation
			_, _, err := net.ParseCIDR(k)
			if err != nil {
				continue
			}

			cache.allowedIPs = append(cache.allowedIPs, AllowedIPEntry{
				Name:   v,
				Subnet: k,
			})
		}
	}
}
