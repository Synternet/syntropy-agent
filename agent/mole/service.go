package mole

import (
	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

func (m *Mole) AddService(si *swireguard.ServiceInfo) error {
	m.Lock()
	defer m.Unlock()
	for _, connectionID := range si.ConnectionIDs {
		m.peers.AddPeerAllowedIps(connectionID, si.IP)
		pi, err := m.peers.GetPeerInfoByConnectionID(connectionID)
		if err != nil {
			return err
		}
		err = m.filter.RulesAdd(si.IP)
		if err != nil {
			logger.Error().Println(pkgName, "iptables rules add", err)
		}

		netpath := &common.SdnNetworkPath{
			Ifname:       pi.IfName,
			PublicKey:    pi.PublicKey,
			ConnectionID: pi.ConnectionID,
			GroupID:      pi.GroupID,
		}

		interfaceCache, _ := m.interfaces.GetInterfaceByIndex(pi.IfIndex)
		netpath.Gateway = interfaceCache.Address

		err = m.wg.AddPeer(pi)
		if err != nil {
			return err
		}

		logger.Debug().Println(pkgName, "Peer service route add to", si.IP,
			"via", netpath.Gateway, netpath.Ifname)
		m.router.RouteAddService(netpath, si.IP)
		if err != nil {
			// Add peer service route failed. It should be some route conflict.
			// In normal case this should not happen.
			// But this is not a fatal error, so I try to warn and continue.
			logger.Warning().Println(pkgName, "adding peer service route", err)
		}
	}
	return nil
}

func (m *Mole) RemoveService(si *swireguard.ServiceInfo) error {
	m.Lock()
	defer m.Unlock()

	for _, connectionID := range si.ConnectionIDs {
		m.peers.AddPeerAllowedIps(connectionID, si.IP)
		pi, err := m.peers.GetPeerInfoByConnectionID(connectionID)
		netpath := &common.SdnNetworkPath{
			Ifname:       pi.IfName,
			PublicKey:    pi.PublicKey,
			ConnectionID: pi.ConnectionID,
			GroupID:      pi.GroupID,
			Gateway:      pi.Gateway,
		}

		interfaceCache, _ := m.interfaces.GetInterfaceByIndex(pi.IfIndex)
		netpath.Gateway = interfaceCache.Address

		err = m.wg.AddPeer(pi)
		if err != nil {
			return err
		}

		// Nobody is interested in RouteDel results
		m.router.RouteDel(netpath, si.IP)
		err = m.filter.RulesDel(si.IP)
		if err != nil {
			logger.Error().Println(pkgName, "iptables rules del", err)
			return err
		}
	}
	return nil
}
