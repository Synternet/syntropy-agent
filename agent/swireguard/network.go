package swireguard

import (
	"net"
	"strconv"

	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

func isPortInRange(port int) bool {
	portStart, portEnd := config.GetPortsRange()
	if portStart == 0 || portEnd == 0 {
		// No ports restriction, if ports range is not configured
		return true
	} else if port >= int(portStart) && port <= int(portEnd) {
		return true
	}
	return false
}

func isFreePort(port int) bool {
	// WG uses UDP for its traffic. Try finding a free UDP port
	l, err := net.ListenPacket("udp", ":"+strconv.Itoa(port))
	if err != nil {
		return false
	}

	l.Close()
	return true
}

func findFreePort(reqPort int) int {
	if isPortInRange(reqPort) && isFreePort(reqPort) {
		return reqPort
	}

	portStart, portEnd := config.GetPortsRange()
	for port := int(portStart); port <= int(portEnd); port++ {
		if isFreePort(port) {
			return port
		}
	}

	// Sad reallity - sometimes you may have no free ports in configured range.
	// Inform user, but try to continue.
	logger.Warning().Println(pkgName, "Could not find free port in configured range. Fallback to random port.")
	return 0
}
