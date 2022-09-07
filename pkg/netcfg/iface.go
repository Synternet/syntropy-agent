// netcfg is a stateless helper to setup interface attributes
// IP, route, interface state, etc
package netcfg

import (
	"fmt"
	"net"
	"net/netip"
	"os"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
)

type RPFilterMode int

const (
	RPFilterNoValidation = iota
	RPFilterStrict
	RPFilterLoose
)

const rpFilterFileFormat = "/proc/sys/net/ipv4/conf/%s/rp_filter"

func setInterfaceRPFilter(ifname string, mode RPFilterMode) error {
	_, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}

	// I could only find this way to set the rp_filter value, it does
	// not seem to have an applicable method via netlink or other syscalls.
	// There is the sysctl syscall, but documentation says it's deprecated
	// and no longer exists on newer kernels, points to using /proc/* system instead.
	proc, err := os.OpenFile(fmt.Sprintf(rpFilterFileFormat, ifname), os.O_WRONLY|os.O_TRUNC, 0611)
	if err != nil {
		return fmt.Errorf("failed to open rp_filter for %v: %v", ifname, err)
	}
	defer proc.Close()

	_, err = proc.WriteString(fmt.Sprintf("%d\n", mode))
	if err != nil {
		return fmt.Errorf("failed to write to rp_filter for %v: %v", ifname, err)
	}

	return nil
}

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

func InterfaceSetRPFilter(ifname string, mode RPFilterMode) error {
	return setInterfaceRPFilter(ifname, mode)
}
