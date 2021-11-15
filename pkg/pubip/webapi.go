package pubip

import (
	"io"
	"net"
	"net/http"
	"strings"
)

func myPublicIp() (net.IP, error) {
	ip := "Unknown" // sane fallback default

	ipProviders := []string{"ip.syntropystack.com:443",
		"ident.me:443",
		"https://ifconfig.me/ip",
		"https://ifconfig.co/ip",
	}
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

	return net.ParseIP(ip), nil
}
