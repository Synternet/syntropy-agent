package mole

import (
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/internal/netfilter"
)

func (m *Mole) CreateInterface(ii *swireguard.InterfaceInfo) error {
	err := m.wg.CreateInterface(ii)

	// Why this config variale configures only forward, and does not impact other iptables rules ???
	if config.CreateIptablesRules() {
		err := netfilter.ForwardEnable(ii.IfName)
		if err != nil {
			logger.Error().Println(pkgName, "netfilter forward enable", ii.IfName, err)
		}
	}

	return err
}

func (m *Mole) RemoveInterface(ii *swireguard.InterfaceInfo) error {
	err := m.wg.RemoveInterface(ii)

	return err
}
