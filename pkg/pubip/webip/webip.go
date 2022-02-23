// Gets public IP address from Syntropy web microservice
package webip

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func PublicIP() (net.IP, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get("https://ip.syntropystack.com:443")
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Could not parse body. Should not happen.
		return nil, err
	}

	// Trim new lines and remove commas
	ipStr := strings.Trim(strings.Trim(string(body), "\n"), "\"")
	ip := net.ParseIP(ipStr)
	if ip == nil || ip.To4() == nil {
		return nil, fmt.Errorf("invalid IP address")
	}

	return ip, nil
}
