// netcfg is a stateless helper to setup interface attributes
// IP, route, interface state, etc
package netcfg

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

func setInterfaceState(ifname string, up bool) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}

	if up {
		return netlink.LinkSetUp(iface)
	} else {
		return netlink.LinkSetDown(iface)
	}
}

func setInterfaceIP(ifname, ip string, add bool) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}

	addr := netlink.Addr{}
	// I think it would be better to have it in CIDR notation
	_, addr.IPNet, _ = net.ParseCIDR(ip)
	if addr.IPNet == nil {
		// But it is plain IP address (with /32 mask in mind)
		addr.IPNet = &net.IPNet{
			IP:   net.ParseIP(ip),
			Mask: net.CIDRMask(32, 32), // TODO: IPv6 support
		}
	}
	if addr.IPNet == nil || addr.IPNet.IP == nil {
		return fmt.Errorf("error parsing IP address %s", ip)
	}

	if add {
		return netlink.AddrAdd(iface, &addr)
	} else {
		return netlink.AddrDel(iface, &addr)
	}
}

func InterfaceUp(ifname string) error {
	return setInterfaceState(ifname, true)
}

func InterfaceDown(ifname string) error {
	return setInterfaceState(ifname, false)
}

func InterfaceIPAdd(ifname, ip string) error {
	return setInterfaceIP(ifname, ip, true)
}

func InterfaceIPDel(ifname, ip string) error {
	return setInterfaceIP(ifname, ip, false)
}
