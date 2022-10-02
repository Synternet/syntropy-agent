package mole

import (
	"strconv"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

func makeKeyFromID(connectionID int) string {
	return strconv.Itoa(connectionID)
}

func (m *Mole) AddService(si *swireguard.ServiceInfo) error {
	m.Lock()
	defer m.Unlock()
	for _, connectionID := range si.ConnectionIDs {
		peer := m.cache.peers[makeKeyFromID(connectionID)]
		err := m.filter.RulesAdd(si.IP)
		if err != nil {
			logger.Error().Println(pkgName, "iptables rules add", err)
		}
		netpath := &common.SdnNetworkPath{
			Ifname:       peer.gwIfname,
			PublicKey:    peer.publicKey,
			ConnectionID: peer.connectionID,
			GroupID:      peer.groupID,
		}
		logger.Debug().Println(pkgName, "Peer host route add to", peer.destIP,
			"via", peer.gateway, peer.gwIfname)
		defaultGw, _, err := netcfg.DefaultRoute()
		err = netcfg.RouteAdd(peer.gwIfname, &defaultGw, &peer.destIP)
		if err != nil {
			// Add peer host route failed. It should be some route conflict.
			// In normal case this should not happen.
			// But this is not a fatal error, so I try to warn and continue.
			logger.Warning().Println(pkgName, "adding peer host route", err)
		}
		err = m.router.RouteAdd(netpath, si.IP)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Mole) RemoveService(si *swireguard.ServiceInfo) error {
	m.Lock()
	defer m.Unlock()

	for _, connectionID := range si.ConnectionIDs {
		peer := m.cache.peers[makeKeyFromID(connectionID)]
		netpath := &common.SdnNetworkPath{
			Ifname:       peer.gwIfname,
			PublicKey:    peer.publicKey,
			ConnectionID: peer.connectionID,
			GroupID:      peer.groupID,
		}
		// Nobody is interested in RouteDel results
		m.router.RouteDel(netpath, si.IP)

		err := m.filter.RulesDel(si.IP)
		if err != nil {
			logger.Error().Println(pkgName, "iptables rules del", err)
			return err
		}
	}
	return nil
}
