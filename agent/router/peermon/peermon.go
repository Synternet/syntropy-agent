package peermon

import (
	"sync"

	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
)

const (
	pkgName = "PeerMonitor. "
	// internal use
	invalidBestIndex = -1
	// For now use simple constant
	// After migration to Go1.18 and netip structures
	// I think to use IsZero()/IsValid() or IsUnspecified() instead
	NoRoute = "no.rou.te"
)

const (
	reasonNewRoute = iota
	reasonLoss
	reasonLatency
)

type PeerMonitor struct {
	sync.RWMutex
	peerList      []*peerInfo
	avgWindowSize uint
	lastBest      int
	changeReason  int
}

func New(avgSize uint) *PeerMonitor {
	return &PeerMonitor{
		lastBest:      invalidBestIndex,
		avgWindowSize: avgSize,
	}
}

func (pm *PeerMonitor) AddNode(ifname, pubKey string, gateway, endpoint string, connID int) {
	pm.Lock()
	defer pm.Unlock()

	for _, peer := range pm.peerList {
		if peer.endpoint == endpoint {
			return
		}
	}

	e := newPeerInfo(pm.avgWindowSize)
	pm.peerList = append(pm.peerList, e)

	e.ifname = ifname
	e.publicKey = pubKey
	e.connectionID = connID
	e.gateway = gateway
	e.endpoint = endpoint
}

func (pm *PeerMonitor) DelNode(endpoint string) {
	pm.Lock()
	defer pm.Unlock()

	for idx, peer := range pm.peerList {
		if peer.endpoint == endpoint {
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

func (pm *PeerMonitor) Peers() []string {
	pm.RLock()
	defer pm.RUnlock()

	rv := []string{}

	for _, peer := range pm.peerList {
		rv = append(rv, peer.endpoint)
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
			pkgName, mark, e.endpoint, e.ifname, e.Latency(), 100*e.Loss())
	}
}

func (pm *PeerMonitor) PingProcess(pr *multiping.PingData) {
	pm.Lock()
	defer pm.Unlock()

	for _, peer := range pm.peerList {
		val, ok := pr.Get(peer.endpoint)
		if !ok {
			// NOTE: PeerMonitor is not creating its own ping list
			// It depends on other pingers and is an additional PingClient in their PingProces line
			// At first it may sound a bit complicate, but in fact it is not.
			// It just looks for its peers in other ping results. And it always founds its peers.
			// NOTE: Do not print error here - PeerMonitor always finds its peers. Just not all of them in one run.
			continue
		}
		peer.Add(val.Latency(), val.Loss())
	}

}

// BestPath returns best route gateway.
// Best route is:
//  * Lowest packet loss
//  * possible lowest latency
// But in order for not to fluctuate between 2 routes, when latency is the same
// so once one best route is found - do not switch to another route, unless it is (betterPercent)% better
func (pm *PeerMonitor) BestPath() string {
	pm.RLock()
	defer pm.RUnlock()

	if len(pm.peerList) == 0 {
		pm.lastBest = invalidBestIndex
		return NoRoute
	}

	// find currently best route
	bestIdx := 0
	for i := bestIdx + 1; i < len(pm.peerList); i++ {
		switch {
		case pm.peerList[i].Loss() > pm.peerList[bestIdx].Loss():
			continue
		case pm.peerList[i].Loss() < pm.peerList[bestIdx].Loss():
			bestIdx = i
		case pm.peerList[i].Latency() > 0 &&
			pm.peerList[i].Latency() < pm.peerList[bestIdx].Latency():
			bestIdx = i
		}
	}

	pm.checkNewBest(bestIdx)

	lossThreshold := config.GetRouteDeleteThreshold()
	if lossThreshold > 0 && pm.peerList[pm.lastBest].Loss()*100 >= float32(lossThreshold) {
		return NoRoute
	}

	return pm.peerList[pm.lastBest].gateway
}

// This function compares currently active best with newly calculated best route
// And compares if route should be changed taken into account route change thresholds
// It changes PeerMonitor's lastBest and changeReason members
func (pm *PeerMonitor) checkNewBest(newIdx int) {
	// No previous best route yet - choose the best
	if pm.lastBest == invalidBestIndex || pm.lastBest >= len(pm.peerList) {
		pm.changeReason = reasonNewRoute
		pm.lastBest = newIdx
		return
	}

	// lower loss is a must
	if pm.peerList[newIdx].Loss() < pm.peerList[pm.lastBest].Loss() {
		pm.changeReason = reasonLoss
		pm.lastBest = newIdx
		return
	}

	// cannot compare latencies, if one does not have full statistics yet
	if pm.peerList[newIdx].StatsIncomplete() {
		return
	}

	// apply thresholds
	diff, ratio := config.RerouteThresholds()
	if pm.peerList[pm.lastBest].Latency()/pm.peerList[newIdx].Latency() >= ratio &&
		pm.peerList[pm.lastBest].Latency()-pm.peerList[newIdx].Latency() >= diff {
		pm.changeReason = reasonLatency
		pm.lastBest = newIdx
	}
}
