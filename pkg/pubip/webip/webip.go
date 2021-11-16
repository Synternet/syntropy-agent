// Gets public IP address from Syntropy web microservice
package webip

import (
	"io"
	"net"
	"net/http"
	"strings"
)

func PublicIP() (net.IP, error) {
	ip := net.ParseIP("0.0.0.0") // sane fallback default

	resp, err := http.Get("https://ip.syntropystack.com:443")
	if err != nil {
		return ip, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Could not parse body. Should not happen.
		return ip, err
	}

	// Trim new lines and remove commas
	ipStr := strings.Trim(strings.Trim(string(body), "\n"), "\"")

	return net.ParseIP(ipStr), nil
}
