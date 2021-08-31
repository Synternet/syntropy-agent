package netcfg

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

func RouteAdd(ifname string, gw string, ips ...string) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %s", ifname)
	}
	gateway := net.ParseIP(gw)

	for _, ip := range ips {
		route := netlink.Route{
			LinkIndex: iface.Attrs().Index,
			Gw:        gateway,
		}
		_, route.Dst, err = net.ParseCIDR(ip)
		if err != nil {
			return fmt.Errorf("%s while parsing %s", err.Error(), ip)
		}
		err = netlink.RouteAdd(&route)
		if err != nil {
			return fmt.Errorf("route %s via %s: %s", ip, gw, err.Error())
		}
	}
	return nil
}

func RouteDel(ifname string, ips ...string) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}

	for _, ip := range ips {
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
	}
	return nil
}

func RouteReplace(ifname string, gw string, ips ...string) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}
	gateway := net.ParseIP(gw)

	for _, ip := range ips {
		route := netlink.Route{
			LinkIndex: iface.Attrs().Index,
			Gw:        gateway,
		}
		_, route.Dst, err = net.ParseCIDR(ip)
		if err != nil {
			return fmt.Errorf("%s while parsing %s", err.Error(), ip)
		}
		err = netlink.RouteAdd(&route)
		if err != nil {
			return fmt.Errorf("route replace %s via %s: %s", ip, gw, err.Error())
		}
	}
	return nil
}

func RouteExists(ip string) bool {
	routes, err := netlink.RouteList(nil, 0)
	if err != nil {
		// Cannot list routes. Should be quite a problem on the system.
		return false
	}
	for _, r := range routes {
		if r.Dst != nil && r.Dst.String() == ip {
			return true
		}
	}
	return false
}
