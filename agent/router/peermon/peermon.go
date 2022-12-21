package peermon

import (
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/peerlist"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/routeselector"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/routeselector/dr"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/routeselector/speed"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping/pingdata"
)

const (
	pkgName = "PeerMonitor. "
)

// PeerMonitr pings configured peers and selects best path
// Implements BestPath() and can be used as PathSelector interface
// PeerMonitor is explicitely used in Router and is always under main Router lock
// So no need for locking here
type PeerMonitor struct {
	peerList *peerlist.PeerList
	config   *routeselector.RouteSelectorConfig
	groupID  int

	pathSelector routeselector.PathSelector
}

func New(cfg *routeselector.RouteSelectorConfig, gid int) *PeerMonitor {
	pm := &PeerMonitor{
		groupID:  gid,
		peerList: peerlist.NewPeerList(cfg.AverageSize),
		config:   cfg,
	}
	if cfg.RouteStrategy == config.RouteStrategyDirectRoute {
		pm.pathSelector = dr.New(pm.peerList, pm.config)
	} else {
		pm.pathSelector = speed.New(pm.peerList, pm.config)
	}

	return pm
}

func (pm *PeerMonitor) AddNode(ifname, pubKey string, endpoint netip.Prefix, connID int, disabled bool) {
	pm.peerList.AddPeer(ifname, pubKey, endpoint, connID, disabled)
}

func (pm *PeerMonitor) DelNode(endpoint netip.Prefix) {
	pm.peerList.DelPeer(endpoint)
}

func (pm *PeerMonitor) HasNode(endpoint netip.Prefix) bool {
	return pm.peerList.HasPeer(endpoint)
}

func (pm *PeerMonitor) Peers() []string {
	return pm.peerList.Peers()
}

func (pm *PeerMonitor) Dump() {
	pm.peerList.Iterate(func(ip netip.Prefix, entry *peerlist.PeerInfo) {
		logger.Debug().Println(pkgName, ip, entry)
	})
}

func (pm *PeerMonitor) Count() int {
	return pm.peerList.Count()
}

func (pm *PeerMonitor) PingProcess(pr *pingdata.PingData) {
	pm.peerList.PingProcess(pr)
}

func (pm *PeerMonitor) BestPath() *routeselector.SelectedRoute {
	return pm.pathSelector.BestPath()
}
