package swireguard

import (
	"math/rand"
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
		// WG uses UDP for its traffic. Try findind a free UDP port
		l, err := net.ListenPacket("udp", ":"+strconv.Itoa(port))
		if err != nil {
			usedPorts[port] = true
			continue
		}

		l.Close()
		return port
	}
}
