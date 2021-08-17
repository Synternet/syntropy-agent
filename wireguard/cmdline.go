package wireguard

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/vishvananda/netlink"
)

func deleteInterface(ifname string) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}

	return netlink.LinkDel(iface)
}

func createInterface(ifname string) error {
	return exec.Command("ip", "link", "add", "dev", ifname, "type", "wireguard").Run()
}

func setInterfaceUp(ifname string) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %v", ifname)
	}

	return netlink.LinkSetUp(iface)
}

func setInterfaceIP(ifname, ip string) error {
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

	return netlink.AddrAdd(iface, &addr)
}
