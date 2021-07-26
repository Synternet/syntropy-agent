package wireguard

import "os/exec"

/**
 TODO: Review these commandlines.
 Port (at least some of them) to native nelink calls.
**/
func deleteInterfaceCmd(ifname string) error {
	return exec.Command("ip", "link", "del", ifname).Run()
}

func createInterfaceCmd(ifname string) error {
	return exec.Command("ip", "link", "add", "dev", ifname, "type", "wireguard").Run()
}

func setInterfaceUpCmd(ifname string) error {
	return exec.Command("ip", "link", "set", "up", ifname).Run()
}

func setInterfaceIPCmd(ifname, ip string) error {
	return exec.Command("ip", "address", "add", "dev", ifname, ip).Run()
}
