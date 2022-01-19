package mole

import (
	"strings"

	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/internal/netfilter"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
	"github.com/SyntropyNet/syntropy-agent/pkg/pubip/webip"
)

func isSdnInterface(ifname string) bool {
	return strings.Contains(ifname, "SDN")
}

func (m *Mole) CreateInterface(ii *swireguard.InterfaceInfo) error {
	m.Lock()
	defer m.Unlock()

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
		err = netcfg.InterfaceSetMTU(ii.IfName, uint32(mtu))
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

	m.cache.ifaces[ii.IfName] = ii.IP

	// If a host is behind NAT - its port after NAT may change.
	// And in most cases this will cause problems for SDN agent.
	// Try detecting NAT and send port as 0 - this way SDN agent will try guessing my port.
	// NOTE: this increases load on SDN agent so use this only when necessary.
	pubip, err := webip.PublicIP()
	if err != nil {
		logger.Error().Println(pkgName, "Error getting public IP", err)
	} else {
		if isSdnInterface(ii.IfName) && !netcfg.HostHasIP(pubip.String()) {
			ii.Port = 0
		}
	}

	// I return nil (no error), because all non-critical errors are already in log
	return nil
}

func (m *Mole) RemoveInterface(ii *swireguard.InterfaceInfo) error {
	m.Lock()
	defer m.Unlock()

	err := m.wg.RemoveInterface(ii)

	delete(m.cache.ifaces, ii.IfName)

	return err
}
