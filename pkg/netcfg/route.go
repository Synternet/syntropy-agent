package netcfg

import (
	"bytes"
	"fmt"
	"net"
	"net/netip"

	"github.com/vishvananda/netlink"
)

func RouteAdd(ifname string, gw *netip.Addr, ip *netip.Prefix) error {
	if ip == nil {
		return fmt.Errorf("no valid IP adress")
	}

	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %s", ifname)
	}

	route := netlink.Route{
		LinkIndex: iface.Attrs().Index,
		Dst: &net.IPNet{
			IP:   ip.Addr().AsSlice(),
			Mask: net.CIDRMask(ip.Bits(), ip.Addr().BitLen()),
		},
	}
	if gw != nil {
		route.Gw = gw.AsSlice()
	}

	if routeExists(iface, route.Dst, route.Gw) {
		// same route already present. Most probably from previous agent instance.
		// It is not error - return success
		return nil
	}

	exists, ifname := RouteSearch(ip)
	if exists {
		return fmt.Errorf("route conflict: %s exists on %s", ip.String(), ifname)
	}

	err = netlink.RouteAdd(&route)
	if err != nil {
		return fmt.Errorf("route %s via %s: %s", ip, gw, err.Error())
	}

	return nil
}

func RouteDel(ifname string, ip *netip.Prefix) error {
	if ip == nil {
		return fmt.Errorf("no valid IP adress")
	}

	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}

	routes, err := netlink.RouteList(iface, 0)
	if err != nil {
		return err
	}
	for _, r := range routes {
		if r.Dst != nil && r.Dst.String() == ip.String() {
			err = netlink.RouteDel(&r)
			if err != nil {
				return fmt.Errorf("route %s del: %s", ip, err.Error())
			}
		} else if r.Dst == nil && IsDefaultRoute(ip) {
			// Deleting default route. Lib returns me nil, but expects Dst to be filled
			r.Dst = &net.IPNet{
				IP:   ip.Addr().AsSlice(),
				Mask: net.CIDRMask(ip.Bits(), ip.Addr().BitLen()),
			}
			err = netlink.RouteDel(&r)
			if err != nil {
				return fmt.Errorf("route %s del: %s", ip, err.Error())
			}
		}
	}

	return nil
}

func RouteReplace(ifname string, gw *netip.Addr, ip *netip.Prefix) error {
	if ip == nil {
		return fmt.Errorf("no valid IP adress")
	}

	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}

	route := netlink.Route{
		LinkIndex: iface.Attrs().Index,
		Dst: &net.IPNet{
			IP:   ip.Addr().AsSlice(),
			Mask: net.CIDRMask(ip.Bits(), ip.Addr().BitLen()),
		},
	}
	if gw != nil {
		route.Gw = gw.AsSlice()
	}

	err = netlink.RouteReplace(&route)
	if err != nil {
		return fmt.Errorf("route replace %s via %s: %s", ip, gw, err.Error())
	}

	return nil
}

func RouteSearch(ip *netip.Prefix) (found bool, ifname string) {
	if ip == nil {
		return false, ""
	}

	routes, err := netlink.RouteList(nil, 0)
	if err != nil {
		// Cannot list routes. Should be quite a problem on the system.
		return
	}
	for _, r := range routes {
		if r.Dst == nil {
			continue
		}
		// We are already listing required interface routes.
		// So need only to compare destination and gateway
		if r.Dst.IP.Equal(ip.Addr().AsSlice()) {
			found = true
			link, err := netlink.LinkByIndex(r.LinkIndex)
			if err == nil {
				ifname = link.Attrs().Name
			}
		}
	}

	return
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
