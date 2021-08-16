package router

import (
	"fmt"
	"log"
	"net"

	"github.com/vishvananda/netlink"
)

func RouteAdd(ifname string, gw string, ips ...string) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}
	gateway := net.ParseIP(gw)

	for _, ip := range ips {
		// TODO: Add IP network overlapping check for all syntropy interfaces
		route := netlink.Route{
			LinkIndex: iface.Attrs().Index,
			Gw:        gateway,
		}
		_, route.Dst, err = net.ParseCIDR(ip)
		if err != nil {
			return fmt.Errorf("%s while parsing %s", err.Error(), ip)
		}
		routes, err := netlink.RouteList(iface, 0)
		if err != nil {
			return err
		}
		dupp := false
		for _, r := range routes {
			if r.Dst != nil && r.Dst.String() == ip && r.Gw.String() == gw {
				log.Printf("Skipping already existing route: %s %s via %s\n",
					ifname, ip, gw)
				dupp = true
				break
			}
		}
		if !dupp {
			err = netlink.RouteAdd(&route)
			log.Println("Route add ", ip, " via ", gw)
			if err != nil {
				return err
			}
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
				log.Println("Route del ", ip)
				if err != nil {
					return err
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
			return err
		}
		log.Println("Route replace ", ip, gw)
	}
	return nil
}
