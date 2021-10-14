package swireguard

import (
	"net"
	"strconv"

	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
)

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
