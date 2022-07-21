package peermon

import (
	"net/netip"
	"sync"

	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const (
	pkgName = "PeerMonitor. "
	// Best route index is not set yet
	invalidBestIndex = -1
)

type SelectedRoute struct {
	IP     *netip.Addr // best route IP address
	ID     int         // ConnectionID of the best route
	Reason *RouteChangeReason
}

type PathSelector interface {
	BestPath() *SelectedRoute
}

type PeerMonitor struct {
	sync.RWMutex
	config   *PeerMonitorConfig
	peerList []*peerInfo
	lastBest int

	pathSelector func(pm *PeerMonitor) (index int, reason *RouteChangeReason)
}

func New(cfg *PeerMonitorConfig) *PeerMonitor {
	pm := &PeerMonitor{
		lastBest: invalidBestIndex,
		config:   cfg,
	}
	if cfg.RouteStrategy == config.RouteStrategyDirectRoute {
		pm.pathSelector = bestPathPreferPublic
	} else {
		pm.pathSelector = bestPathLowestLatency
	}

	return pm
}

func (pm *PeerMonitor) AddNode(ifname, pubKey string, endpoint netip.Addr, connID int) {
	pm.Lock()
	defer pm.Unlock()

	for _, peer := range pm.peerList {
		if peer.ip == endpoint {
			return
		}
	}

	e := newPeerInfo(pm.config.AverageSize)
	pm.peerList = append(pm.peerList, e)

	e.ifname = ifname
	e.publicKey = pubKey
	e.connectionID = connID
	e.ip = endpoint
}

func (pm *PeerMonitor) DelNode(endpoint netip.Addr) {
	pm.Lock()
	defer pm.Unlock()

	for idx, peer := range pm.peerList {
		if peer.ip == endpoint {
			// order is not important.
			// Remove from slice in more effective way
			pm.peerList[idx] = pm.peerList[len(pm.peerList)-1]
			pm.peerList = pm.peerList[:len(pm.peerList)-1]
			// Check and invalidate last best path index
			if idx == pm.lastBest {
				pm.lastBest = invalidBestIndex
			}
			return
		}
	}
}

func (pm *PeerMonitor) HasNode(endpoint netip.Addr) bool {
	pm.Lock()
	defer pm.Unlock()

	for _, peer := range pm.peerList {
		if peer.ip == endpoint {
			return true
		}
	}
	return false
}

func (pm *PeerMonitor) Peers() []string {
	pm.RLock()
	defer pm.RUnlock()

	rv := []string{}

	for _, peer := range pm.peerList {
		rv = append(rv, peer.ip.String())
	}
	return rv
}

func (pm *PeerMonitor) Dump() {
	for i := 0; i < len(pm.peerList); i++ {
		e := pm.peerList[i]
		mark := " "
		if i == pm.lastBest {
			mark = "*"
		}
		logger.Debug().Printf("%s%s %s\t%s\t%fms\t%f%%\n",
			pkgName, mark, e.ip.String(), e.ifname, e.Latency(), 100*e.Loss())
	}
}

func (pm *PeerMonitor) Close() error {
	// nothing to do in peer monitor yet
	// All peer routes will be deleted once interface is deleted
	return nil
}

func (pm *PeerMonitor) Flush() {
	// nothing to do in peer monitor yet
	// All peer routes will be deleted once interface is deleted
}
