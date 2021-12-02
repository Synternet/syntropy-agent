package mole

import (
	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
)

func makeKey(pi *swireguard.PeerInfo) string {
	return pi.IfName + pi.PublicKey
}

func (m *Mole) AddPeer(pi *swireguard.PeerInfo, netpath *common.SdnNetworkPath) error {
	err := m.wg.AddPeer(pi)
	if err != nil {
		return err
	}

	m.cache[makeKey(pi)] = entry{
		groupID:      pi.GroupID,
		connectionID: pi.ConnectionID,
	}

	return m.router.RouteAdd(netpath, pi.AllowedIPs...)
}

func (m *Mole) RemovePeer(pi *swireguard.PeerInfo, netpath *common.SdnNetworkPath) error {
	// Nobody is interested in RouteDel results
	m.router.RouteDel(netpath, pi.AllowedIPs...)

	delete(m.cache, makeKey(pi))

	return m.wg.RemovePeer(pi)
}
