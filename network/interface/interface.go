// This package needs a good review ASAP
// Only draft POC here now
package netiface

import "os/exec"

/**
 TODO: thing about:
  * moving this to separate package,
  * native netlink implementation
          TODO: refactor this 100% :TODO
**/
func DeleteInterfaceCmd(ifname string) error {
	return exec.Command("ip", "link", "del", ifname).Run()
}

func CreateInterfaceCmd(ifname string) error {
	return exec.Command("ip", "link", "add", "dev", ifname, "type", "wireguard").Run()
}

func SetInterfaceUpCmd(ifname string) error {
	return exec.Command("ip", "link", "set", "up", ifname).Run()
}

func SetInterfaceIPCmd(ifname, ip string) error {
	return exec.Command("ip", "address", "add", "dev", ifname, ip).Run()
}
