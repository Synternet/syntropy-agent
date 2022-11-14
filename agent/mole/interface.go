package mole

import (
	"net/netip"
	"strings"

	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
	"github.com/SyntropyNet/syntropy-agent/pkg/pubip"
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
		// Note: thats one of critical errors
		return err
	}

	err = netcfg.InterfaceUp(ii.IfName)
	if err != nil {
		logger.Error().Println(pkgName, "Could not up interface: ", ii.IfName, err)
	}
	err = netcfg.InterfaceIPSet(ii.IfName, ii.IP)
	if err != nil {
		logger.Error().Println(pkgName, "Could not set IP address: ", ii.IfName, err)
	}

	if mtu := config.GetInterfaceMTU(); mtu > 0 {
		err = netcfg.InterfaceSetMTU(ii.IfName, uint32(mtu))
		if err != nil {
			logger.Error().Println(pkgName, "MTU error: ", ii.IfName, mtu, err)
		}
	}

	// Why this config variale configures only forward, and does not impact other iptables rules ???
	if config.CreateIptablesRules() {
		err = m.filter.ForwardEnable(ii.IfName)
		if err != nil {
			logger.Error().Println(pkgName, "netfilter forward enable", ii.IfName, err)
		}
	}

	// Recheck interfaces PublicKey and Port after interface is set up
	// This function also updates cache
	err = m.wg.CheckInterface(ii)
	if err != nil {
		logger.Error().Println(pkgName, "check interface", err)
	}

	m.cache.ifaces[ii.IfName] = ii.IP

	// If a host is behind NAT - its port after NAT may change.
	// And in most cases this will cause problems for SDN agent.
	// Try detecting NAT and send port as 0 - this way SDN agent will try guessing my port.
	// NOTE: this increases load on SDN agent so use this only when necessary.
	if isSdnInterface(ii.IfName) {
		publicIP := pubip.GetPublicIp()
		if !publicIP.IsUnspecified() {
			pubip, ok := netip.AddrFromSlice(publicIP)
			if ok && !netcfg.HostHasIP(pubip) {
				ii.Port = 0
			}
		} else {
			logger.Error().Println(pkgName, "Error getting public IP")
			// Could not get public IP - thus cannod detect if NAT is present
			// Give more work for SDN agent and tell him to detect the port
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

func (m *Mole) CreateChain() error {
	return m.filter.CreateChain()
}
