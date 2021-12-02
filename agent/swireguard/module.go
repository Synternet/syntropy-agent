package swireguard

import (
	"bufio"
	"os"
	"strings"

	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"pault.ag/go/modprobe"
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

func loadKernelModule() {
	if isKernelModuleLoaded() {
		return
	}

	// Sometimes pault.ag/modprobe package crashes.
	// I think it happens when distribution kernel was upgraded, but OS not yet rebooted
	// And package is trying to load different kernels module
	// So add a possible recover function here
	defer func() {
		if err := recover(); err != nil {
			// I intentionally do not print error message here.
			// Because I did a recover and will fallback to userspace WG implementation
			// And those error messages usuly sound very scary and may insult users.
			logger.Error().Println(pkgName, "error loading kernel module (an OS reboot may be required)")
		}
	}()

	err := modprobe.Load("wireguard", "")
	if err != nil {
		logger.Warning().Println(pkgName, "kernel module load", err)
	}
}

func (wg *Wireguard) LogInfo() {
	if isKernelModuleLoaded() {
		logger.Info().Println(pkgName, "Using Wireguard implementation in-kernel")
	} else {
		logger.Info().Println(pkgName, "Using userspace Wireguard implementation `wireguard-go`")
	}
}
