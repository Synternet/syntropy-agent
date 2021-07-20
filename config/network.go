package config

import (
	"io"
	"net/http"
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
