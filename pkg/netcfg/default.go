package netcfg

import (
	"errors"
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

func DefaultRoute() (string, string, error) {
	var defaultRoute *netlink.Route
	var ifname string
	var err error

	routes, err := netlink.RouteList(nil, unix.AF_INET)
	if err != nil {
		return "", "", err
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
		return "", "", ErrNotFound
	}

	ifname, err = ifnameFromIndex(defaultRoute.LinkIndex)
	return defaultRoute.Gw.String(), ifname, err
}
