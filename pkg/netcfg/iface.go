// netcfg is a stateless helper to setup interface attributes
// IP, route, interface state, etc
package netcfg

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
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

func setInterfaceIP(ifname string, ip netip.Addr, add bool) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}

	addr := netlink.Addr{
		IPNet: &net.IPNet{
			IP:   ip.AsSlice(),
			Mask: net.CIDRMask(ip.BitLen(), ip.BitLen()),
		},
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

// InterfaceIPAdd adds `ip` to interface `ifname`
func InterfaceIPAdd(ifname string, ip netip.Addr) error {
	return setInterfaceIP(ifname, ip, true)
}

// InterfaceIPDel removes `ip` from interface `ifname`
func InterfaceIPDel(ifname string, ip netip.Addr) error {
	return setInterfaceIP(ifname, ip, false)
}

// InterfaceIPSet removes old IP addresses from interface `ifname`
// and sets `ip` as the only address
func InterfaceIPSet(ifname string, ip netip.Addr) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}
	exists := false

	// First remove residual old addresses
	ifaceAddrs, _ := netlink.AddrList(iface, nl.FAMILY_ALL)
	for _, addr := range ifaceAddrs {
		if addr.IP.Equal(ip.AsSlice()) {
			exists = true
			continue
		}
		netlink.AddrDel(iface, &addr)
	}

	// If address is already set - do not set it once more
	if exists {
		return nil
	}

	// Address is missing - set it
	addr := netlink.Addr{
		IPNet: &net.IPNet{
			IP:   ip.AsSlice(),
			Mask: net.CIDRMask(ip.BitLen(), ip.BitLen()),
		},
	}

	return netlink.AddrAdd(iface, &addr)
}

func InterfaceHasIP(ifname string, ipAddress netip.Addr) bool {
	ip := ipAddress.AsSlice()

	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return false
	}

	ifaceAddrs, _ := netlink.AddrList(iface, nl.FAMILY_ALL)
	for _, addr := range ifaceAddrs {
		if addr.IP.Equal(ip) {
			return true
		}
	}
	return false
}

func HostHasIP(ipAddress netip.Addr) bool {
	ip := ipAddress.AsSlice()
	ifaceAddrs, _ := netlink.AddrList(nil, nl.FAMILY_ALL)
	for _, addr := range ifaceAddrs {
		if addr.IP.Equal(ip) {
			return true
		}
	}
	return false
}

func InterfaceSetMTU(ifname string, mtu uint32) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return err
	}

	return netlink.LinkSetMTU(iface, int(mtu))
}
