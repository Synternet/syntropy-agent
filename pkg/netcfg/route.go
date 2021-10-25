package netcfg

import (
	"bytes"
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

func RouteAdd(ifname string, gw string, ip string) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %s", ifname)
	}
	gateway := net.ParseIP(gw)

	route := netlink.Route{
		LinkIndex: iface.Attrs().Index,
		Gw:        gateway,
	}

	// I need only network address here, no need to "patch" parseCidr's result
	_, route.Dst, err = net.ParseCIDR(ip)
	if err != nil {
		return fmt.Errorf("%s while parsing %s", err.Error(), ip)
	}

	if routeExists(iface, route.Dst, route.Gw) {
		// same route already present. Most probably from previous agent instance.
		// It is not error - return success
		return nil
	}

	err = checkRouteConflicts(route.Dst)
	if err != nil {
		return err
	}

	err = netlink.RouteAdd(&route)
	if err != nil {
		return fmt.Errorf("route %s via %s: %s", ip, gw, err.Error())
	}

	return nil
}

func RouteDel(ifname string, ip string) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}

	routes, err := netlink.RouteList(iface, 0)
	if err != nil {
		return err
	}
	for _, r := range routes {
		if r.Dst != nil && r.Dst.String() == ip {
			err = netlink.RouteDel(&r)
			if err != nil {
				return fmt.Errorf("route %s del: %s", ip, err.Error())
			}
		}
	}

	return nil
}

func RouteReplace(ifname string, gw string, ip string) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}
	gateway := net.ParseIP(gw)

	route := netlink.Route{
		LinkIndex: iface.Attrs().Index,
		Gw:        gateway,
	}

	// I need only network address here, no need to "patch" parseCidr's result
	_, route.Dst, err = net.ParseCIDR(ip)
	if err != nil {
		return fmt.Errorf("%s while parsing %s", err.Error(), ip)
	}
	err = netlink.RouteReplace(&route)
	if err != nil {
		return fmt.Errorf("route replace %s via %s: %s", ip, gw, err.Error())
	}

	return nil
}

func routeExists(link netlink.Link, dst *net.IPNet, gw net.IP) bool {
	if dst == nil {
		return true // TODO: default route checking. But now we do not configure default routes
	}

	routes, err := netlink.RouteList(link, 0)
	if err != nil {
		// Cannot list routes. Should be quite a problem on the system.
		return false
	}
	for _, r := range routes {
		if r.Dst == nil {
			continue
		}
		// We are already listing required interface routes.
		// So need only to compare destination and gateway
		if r.Dst.IP.Equal(dst.IP) &&
			bytes.Equal(r.Dst.Mask, dst.Mask) && r.Gw.Equal(gw) {
			return true
		}
	}
	return false
}

func checkRouteConflicts(dst *net.IPNet) error {
	if dst == nil {
		return nil // we can have several default routes, right
	}
	routes, err := netlink.RouteList(nil, 0)
	if err != nil {
		// Cannot list routes. Should be quite a problem on the system.
		return err
	}

	for _, r := range routes {
		if r.Dst == nil {
			continue
		}
		// In this case do not worry about correct rule already existing.
		// This case was already checked in `routeExists()`, and code should never reach here
		// If I am here - then we have a dupplicate route.
		// Error out and do not add additional conflict route
		if r.Dst.String() == dst.String() {
			return fmt.Errorf("route conflict %s vs %s [%d]", dst.String(), r.Dst.String(), r.LinkIndex)
		}
	}
	return nil
}
