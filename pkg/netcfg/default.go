package netcfg

import (
	"errors"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

var ErrNotFound = errors.New("default route not found")

func DefaultRoute() (string, string, error) {
	var defaultRoute *netlink.Route

	routes, err := netlink.RouteList(nil, unix.AF_INET)
	if err != nil {
		return "", "", err
	}

	for idx, r := range routes {
		if r.Dst == nil {
			if defaultRoute == nil || defaultRoute.Priority > r.Priority {
				defaultRoute = &routes[idx]
			}

		}
	}

	if defaultRoute == nil {
		return "", "", ErrNotFound
	}

	l, err := netlink.LinkByIndex(defaultRoute.LinkIndex)
	if err != nil {
		return "", "", err
	}

	return defaultRoute.Gw.String(), l.Attrs().Name, nil
}
