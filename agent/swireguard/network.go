package swireguard

import (
	"math/rand"
	"net"
	"strconv"
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
)

func isBehindNAT() bool {
	// TODO: implement me
	return true
}

func isSDN(ifname string) bool {
	return strings.Contains(ifname, "SDN")
}

func GetFreePort(ifname string) int {
	if isSDN(ifname) && isBehindNAT() {
		return 0
	}

	portStart, portEnd := config.GetPortsRange()
	usedPorts := make(map[int]bool)

	for {
		port := rand.Intn(int(portEnd-portStart)) + int(portStart)

		// skip previously checked ports
		if _, ok := usedPorts[port]; ok {
			continue
		}

		// TODO: I'm pretty sure WG uses UDP for its traffic
		// Improove the free ports check
		l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			usedPorts[port] = true
			continue
		}

		l.Close()
		return port
	}
}
