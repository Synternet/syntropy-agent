package peermon

import (
	"sync"

	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
)

const (
	pkgName = "PeerMonitor. "
	// how many values use for moving average
	// this value is like multiplicator for peerdata.periodRun
	// if peerdata.periodRun=5secs, then 5*24=2 minutes average
	valuesCount = 24
	// when have an active route and want to change a better route
	// this new better route must be 10% better, to reduce route change fluctuation
	betterCoeficient = 0.9
	// internal use
	invalidBestIndex = -1
)

const (
	reasonNewRoute = iota
	reasonLoss
	reasonLatency
)

type PeerMonitor struct {
	sync.RWMutex
	peerList     []*peerInfo
	lastBest     int
	changeReason int
}

func New() *PeerMonitor {
	return &PeerMonitor{
		lastBest: invalidBestIndex,
	}
}

func (pm *PeerMonitor) AddNode(gateway, endpoint string) {
	pm.Lock()
	defer pm.Unlock()

	for _, peer := range pm.peerList {
		if peer.endpoint == endpoint {
			return
		}
	}

	e := peerInfo{
		gateway:  gateway,
		endpoint: endpoint,
	}
	pm.peerList = append(pm.peerList, &e)
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
			pkgName, mark, e.endpoint, e.gateway, e.Latency(), 100*e.Loss())
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
		return ""
	}

	// find currently best route
	bestIdx := 0
	for i := bestIdx + 1; i < len(pm.peerList); i++ {
		if pm.peerList[i].Loss() < pm.peerList[bestIdx].Loss() {
			bestIdx = i
		} else if pm.peerList[i].Latency() > 0 &&
			pm.peerList[i].Latency() < pm.peerList[bestIdx].Latency() {
			bestIdx = i
		}
	}

	pm.checkNewBest(bestIdx)

	return pm.peerList[pm.lastBest].gateway
}

func (pm *PeerMonitor) checkNewBest(idx int) {
	if pm.lastBest == invalidBestIndex {
		// No previous best route yet - choose the best
		pm.changeReason = reasonNewRoute
		pm.lastBest = idx
	} else {
		switch {
		case pm.peerList[idx].Loss() < pm.peerList[pm.lastBest].Loss():
			// lower loss is a must
			pm.changeReason = reasonLoss
			pm.lastBest = idx
		case pm.peerList[idx].Latency() < pm.peerList[pm.lastBest].Latency()*betterCoeficient:
			// reduce too much reroutes and move to other route only if it is xx% better
			pm.changeReason = reasonLatency
			pm.lastBest = idx
		}
	}
}
