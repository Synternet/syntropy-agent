package swireguard

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
)

func isKernelModuleLoaded() bool {
	// parse /proc/module file and search for `wireguard`
	file, err := os.Open("/proc/modules")
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "wireguard") {
			return true
		}
	}

	return false
}

func loadKernelModule() error {
	if isKernelModuleLoaded() {
		return nil
	}

	return exec.Command("modprobe", "wireguard").Run()
}
