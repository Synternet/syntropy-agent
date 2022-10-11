package peermon

import (
	"net/netip"
	"sync"

	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const (
	pkgName = "PeerMonitor. "
)

// Best route index is not set yet
var invalidBest = netip.Prefix{}

type SelectedRoute struct {
	IP     netip.Addr // best route IP address
	ID     int        // ConnectionID of the best route
	Reason *RouteChangeReason
}

type PathSelector interface {
	BestPath() *SelectedRoute
}

type PeerMonitor struct {
	sync.RWMutex
	config   *PeerMonitorConfig
	peerList map[netip.Prefix]*peerInfo
	lastBest netip.Prefix

	pathSelector func(pm *PeerMonitor) (addr netip.Prefix, reason *RouteChangeReason)
}

func New(cfg *PeerMonitorConfig) *PeerMonitor {
	pm := &PeerMonitor{
		peerList: make(map[netip.Prefix]*peerInfo),
		lastBest: invalidBest,
		config:   cfg,
	}
	if cfg.RouteStrategy == config.RouteStrategyDirectRoute {
		pm.pathSelector = bestPathPreferPublic
	} else {
		pm.pathSelector = bestPathLowestLatency
	}

	return pm
}

func (pm *PeerMonitor) AddNode(ifname, pubKey string, endpoint netip.Prefix, connID int) {
	pm.Lock()
	defer pm.Unlock()

	e, ok := pm.peerList[endpoint]
	if !ok {
		e = newPeerInfo(pm.config.AverageSize)
		pm.peerList[endpoint] = e
	}

	e.ifname = ifname
	e.publicKey = pubKey
	e.connectionID = connID
}

func (pm *PeerMonitor) DelNode(endpoint netip.Prefix) {
	pm.Lock()
	defer pm.Unlock()

	// Check and invalidate last best path index
	if pm.lastBest == endpoint {
		pm.lastBest = invalidBest
	}

	_, ok := pm.peerList[endpoint]
	if ok {
		delete(pm.peerList, endpoint)
	}
}

func (pm *PeerMonitor) HasNode(endpoint netip.Prefix) bool {
	pm.Lock()
	defer pm.Unlock()

	_, ok := pm.peerList[endpoint]

	return ok
}

func (pm *PeerMonitor) Peers() []string {
	pm.RLock()
	defer pm.RUnlock()

	rv := []string{}

	for ip, _ := range pm.peerList {
		rv = append(rv, ip.String())
	}
	return rv
}

func (pm *PeerMonitor) Dump() {
	for ip, e := range pm.peerList {
		mark := " "
		if pm.lastBest == ip {
			mark = "*"
		}
		logger.Debug().Println(pkgName, mark, ip, e)
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
