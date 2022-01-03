package mole

import (
	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
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

	cacheEntry := peerCacheEntry{
		groupID:      pi.GroupID,
		connectionID: pi.ConnectionID,
		// TODO: This quite ugly hack will be fixed upon migration to Go1.18
		// and using netip struct instead of string
		destIP: pi.IP + "/32",
	}

	defaultGw, defaultIfname, err := netcfg.DefaultRoute()
	if err == nil {
		cacheEntry.gateway = defaultGw
		cacheEntry.gwIfname = defaultIfname
		logger.Debug().Println(pkgName, "Peer host route add to", cacheEntry.destIP,
			"via", cacheEntry.gateway, cacheEntry.gwIfname)
		err = netcfg.RouteAdd(cacheEntry.gwIfname, cacheEntry.gateway, cacheEntry.destIP)
		if err != nil {
			// Add peer host route failed. It should be some route conflict.
			// In normal case this should not happen.
			// But this is not a fatal error, so I try to warn and continue.
			logger.Warning().Println(pkgName, "adding peer host route", err)
		}
	} else {
		logger.Warning().Println(pkgName, "could not parse default route")
	}

	m.cache.peers[makeKey(pi)] = cacheEntry
	return m.router.RouteAdd(netpath, pi.AllowedIPs...)
}

func (m *Mole) RemovePeer(pi *swireguard.PeerInfo, netpath *common.SdnNetworkPath) error {
	m.Lock()
	defer m.Unlock()

	cacheKey := makeKey(pi)
	entry, ok := m.cache.peers[cacheKey]
	// Controller does not send Connection and Group IDs in packet.
	// Need to find them in cache
	if ok {
		netpath.ConnectionID = entry.connectionID
		netpath.GroupID = entry.groupID

		if entry.gwIfname != "" {
			logger.Debug().Println(pkgName, "Peer host route del to", entry.destIP,
				"via", entry.gwIfname)
			err := netcfg.RouteDel(entry.gwIfname, entry.destIP)
			if err != nil {
				// Host route deletion failed.
				// Most probably network configuration has changed.
				// P.S. This is not a fatal error. Warning and try to continue.
				logger.Warning().Println(pkgName, "peer host route delete", err)
			}
		}
	}

	// Same is with interface IP address
	netpath.Gateway, ok = m.cache.ifaces[pi.IfName]
	if !ok {
		logger.Warning().Println(pkgName, pi.IfName, "not found in cache")
	}

	// Nobody is interested in RouteDel results
	m.router.RouteDel(netpath, pi.AllowedIPs...)

	delete(m.cache.peers, cacheKey)

	return m.wg.RemovePeer(pi)
}
