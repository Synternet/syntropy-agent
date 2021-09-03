package wireguard

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
)

func IsKernelModuleLoaded() bool {
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

func LoadKernelModule() {
	exec.Command("modprobe", "wireguard").Run()
}
