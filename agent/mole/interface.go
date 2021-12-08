package mole

import (
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/internal/netfilter"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

func (m *Mole) CreateInterface(ii *swireguard.InterfaceInfo) error {
	err := m.wg.CreateInterface(ii)

	if err != nil {
		logger.Error().Println(pkgName, "create interface", err)
		// Note: thats the only critical error
		return err
	}

	// Why this config variale configures only forward, and does not impact other iptables rules ???
	if config.CreateIptablesRules() {
		err = netfilter.ForwardEnable(ii.IfName)
		if err != nil {
			logger.Error().Println(pkgName, "netfilter forward enable", ii.IfName, err)
		}
	}

	if mtu := config.GetInterfaceMTU(); mtu > 0 {
		err = netcfg.InterfaceSetMTU(ii.IfName, mtu)
		if err != nil {
			logger.Error().Println(pkgName, "MTU error: ", ii.IfName, mtu, err)
		}
	}

	err = netcfg.InterfaceUp(ii.IfName)
	if err != nil {
		logger.Error().Println(pkgName, "Could not up interface: ", ii.IfName, err)
	}
	err = netcfg.InterfaceIPSet(ii.IfName, ii.IP)
	if err != nil {
		logger.Error().Println(pkgName, "Could not set IP address: ", ii.IfName, err)
	}

	// I return nil (no error), because all non-critical errors are already in log
	return nil
}

func (m *Mole) RemoveInterface(ii *swireguard.InterfaceInfo) error {
	err := m.wg.RemoveInterface(ii)

	return err
}
