package mole

import (
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

func (m *Mole) AddPeer(pi *swireguard.PeerInfo, netpath *common.SdnNetworkPath) error {
	m.Lock()
	defer m.Unlock()
	err := m.wg.AddPeer(pi)
	if err != nil {
		return err
	}

	err = m.filter.RulesAdd(pi.AllowedIPs...)
	if err != nil {
		logger.Error().Println(pkgName, "iptables rules add", err)
	}

	m.peers.Add(pi)
	// Add a single host route address
	err = m.hostRoute.Add(netip.PrefixFrom(pi.IP, pi.IP.BitLen()))
	if err != nil {
		logger.Error().Println(pkgName, "host route add", err)
	}

	interfaceCache, _ := m.interfaces.GetInterfaceByIndex(pi.IfIndex)
	netpath.Gateway = interfaceCache.Address

	return m.router.RouteAdd(netpath, pi.AllowedIPs...)
}

func (m *Mole) RemovePeer(pi *swireguard.PeerInfo, netpath *common.SdnNetworkPath) error {
	m.Lock()
	defer m.Unlock()

	// check connection ID and GID in cache
	err := m.peers.CheckAndDel(pi)
	if err != nil {
		logger.Error().Println(pkgName, "peer cache check", err)
	} else {
		netpath.ConnectionID = pi.ConnectionID
		netpath.GroupID = pi.GroupID
	}

	// Delete a single host route address
	err = m.hostRoute.Del(netip.PrefixFrom(pi.IP, pi.IP.BitLen()))
	if err != nil {
		logger.Error().Println(pkgName, "host route add", err)
	}

	// Nobody is interested in RouteDel results
	m.router.RouteDel(netpath, pi.AllowedIPs...)

	err = m.filter.RulesDel(pi.AllowedIPs...)
	if err != nil {
		logger.Error().Println(pkgName, "iptables rules del", err)
	}

	return m.wg.RemovePeer(pi)
}
