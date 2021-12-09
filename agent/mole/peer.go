package mole

import (
	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

func makeKey(pi *swireguard.PeerInfo) string {
	return pi.IfName + pi.PublicKey
}

func (m *Mole) AddPeer(pi *swireguard.PeerInfo, netpath *common.SdnNetworkPath) error {
	m.Lock()
	defer m.Unlock()

	err := m.wg.AddPeer(pi)
	if err != nil {
		return err
	}

	m.cache.peers[makeKey(pi)] = peerIDs{
		groupID:      pi.GroupID,
		connectionID: pi.ConnectionID,
	}

	return m.router.RouteAdd(netpath, pi.AllowedIPs...)
}

func (m *Mole) RemovePeer(pi *swireguard.PeerInfo, netpath *common.SdnNetworkPath) error {
	m.Lock()
	defer m.Unlock()

	entry, ok := m.cache.peers[makeKey(pi)]
	// Controller does not send Connection and Group IDs in packet.
	// Need to find them in cache
	if ok {
		netpath.ConnectionID = entry.connectionID
		netpath.GroupID = entry.groupID
	}
	// Same is with interface IP address
	netpath.Gateway, ok = m.cache.ifaces[pi.IfName]
	if !ok {
		logger.Warning().Println(pkgName, pi.IfName, "not found in cache")
	}

	// Nobody is interested in RouteDel results
	m.router.RouteDel(netpath, pi.AllowedIPs...)

	delete(m.cache.peers, makeKey(pi))

	return m.wg.RemovePeer(pi)
}
