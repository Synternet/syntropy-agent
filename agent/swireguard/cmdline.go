package swireguard

import (
	"fmt"
	"os/exec"

	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/vishvananda/netlink"
)

func (wg *Wireguard) deleteInterface(ifname string) error {
	iface, err := netlink.LinkByName(ifname)
	if err != nil {
		return fmt.Errorf("failed to lookup interface %s", ifname)
	}

	// ip link del dev <interface> works for both - kernel and userspace WG implementations.
	return netlink.LinkDel(iface)
}

func (wg *Wireguard) createInterface(ifname string) error {
	// XXX vishvananda netlink package is not (yet) capable of creating wireguard interface type
	err := exec.Command("ip", "link", "add", "dev", ifname, "type", "wireguard").Run()
	if err != nil {
		logger.Warning().Println(pkgName, "Could not create kernel wireguard interface: ", err)
		err = exec.Command("wireguard-go", ifname).Run()
	}
	return err
}
