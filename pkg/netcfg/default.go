package netcfg

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

var ErrNotFound = errors.New("default route not found")

func ifnameFromIndex(idx int) (string, error) {
	l, err := netlink.LinkByIndex(idx)
	if err != nil {
		return "", err
	}

	return l.Attrs().Name, nil
}

// IsDefaultRoute returns true if addr == 0.0.0.0/0
func IsDefaultRoute(addr *netip.Prefix) bool {
	return addr.Addr().IsUnspecified() && addr.Bits() == 0
}

func DefaultRoute() (netip.Addr, string, error) {
	var defaultRoute *netlink.Route
	var ifname string
	var err error

	routes, err := netlink.RouteList(nil, unix.AF_INET)
	if err != nil {
		return netip.IPv4Unspecified(), "", err
	}

	for idx, r := range routes {
		// In VPN case SYNTROPY_ interface can already be added as default route
		// Ignore them and try finding real default route
		ifname, _ = ifnameFromIndex(r.LinkIndex)
		if strings.Contains(ifname, env.InterfaceNamePrefix) {
			continue
		}

		if r.Dst == nil {
			if defaultRoute == nil || defaultRoute.Priority > r.Priority {
				defaultRoute = &routes[idx]
			}

		}
	}

	if defaultRoute == nil {
		return netip.IPv4Unspecified(), "", ErrNotFound
	}

	ifname, err = ifnameFromIndex(defaultRoute.LinkIndex)
	if err != nil {
		return netip.IPv4Unspecified(), "", err
	}
	addr, ok := netip.AddrFromSlice(defaultRoute.Gw)
	if !ok {
		return netip.IPv4Unspecified(), "", fmt.Errorf("Failed parsing IP address %s", defaultRoute.Gw)
	}

	return addr, ifname, nil
}
