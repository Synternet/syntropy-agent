package swireguard

import (
	"net"
	"strconv"
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
)

func isBehindNAT() bool {
	// List all OS IP addresses and compare it with a public IP
	publicIP := config.GetPublicIp()
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Error().Println(pkgName, "listing IP address", err)
		return true
	}

	for _, a := range addrs {
		// net.Addr.String() shows local address as 192.168.1.2/24
		// Remove the mask
		v := strings.Split(a.String(), "/")
		if len(v) > 0 && v[0] == publicIP {
			return false
		}
	}

	// Seems I do not have public IP address on my interfaces.
	// This means host machine is behind NAT
	return true
}

func isSDN(ifname string) bool {
	return strings.Contains(ifname, "SDN")
}

func GetValidPort(reqPort int) int {
	portStart, portEnd := config.GetPortsRange()

	isFreePort := func(p int) bool {
		// WG uses UDP for its traffic. Try finding a free UDP port
		l, err := net.ListenPacket("udp", ":"+strconv.Itoa(p))
		if err != nil {
			return false
		}

		l.Close()
		return true
	}

	// No ports restriction, if ports range is not configured
	if portStart == 0 || portEnd == 0 {
		return reqPort
	}

	if reqPort < int(portStart) || reqPort > int(portEnd) {
		reqPort = int(portStart)
	}

	// First try requested port, then check if any port in configured range is available
	for port := reqPort; port <= int(portEnd); port++ {
		if isFreePort(port) {
			return port
		}
	}
	// the other part of range
	for port := int(portStart); port < reqPort; port++ {
		if isFreePort(port) {
			return port
		}
	}

	// Sad reallity - sometimes you may have no free ports in configured range.
	// Inform user, but try to continue.
	logger.Warning().Println(pkgName, "Could not find free port in configured range. Fallback to random port.")
	return 0
}
