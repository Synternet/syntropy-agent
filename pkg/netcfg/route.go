package netcfg

import (
	"fmt"
	"net"

	// TODO: this helper package should not use logger
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/vishvananda/netlink"
)

const pkgName = "NetCfg. "

func RouteAdd(ifname string, gw string, ips ...string) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %s", ifname)
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
		routes, err := netlink.RouteList(nil, 0)
		if err != nil {
			return err
		}
		dupp := false
		for _, r := range routes {
			if r.Dst != nil && r.Dst.String() == ip {
				logger.Debug().Printf("%s Skipping already existing route: %s %s via %s\n",
					pkgName, ifname, ip, gw)
				dupp = true
				break
			}
		}
		if !dupp {
			err = netlink.RouteAdd(&route)
			logger.Info().Println(pkgName, "Route add ", ip, " via ", gw)
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
				logger.Info().Println(pkgName, "Route del ", ip)
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
		logger.Info().Println("Route replace ", ip, gw)
	}
	return nil
}
