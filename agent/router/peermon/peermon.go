package peermon

// TODO: think about merging this with router.Router and using some interfaces

import (
	"sync"

	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
)

const (
	pkgName          = "PeerMonitor. "
	valuesCount      = 10  // how much values use for moving average
	betterCoeficient = 0.9 // 10% better
	invalidBestIndex = -1
)

type PeerMonitor struct {
	sync.RWMutex
	peerList []*peerInfo
	lastBest int
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

func (pm *PeerMonitor) PingProcess(pr []multiping.PingResult) {
	pm.Lock()
	defer pm.Unlock()

	for _, res := range pr {
		for _, peer := range pm.peerList {
			if peer.endpoint == res.IP {
				peer.Add(res.Latency, res.Loss)
				break // break internal loop, continue on external
			}
		}
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

	if pm.lastBest == invalidBestIndex {
		// No previous best route yet - choose the best
		pm.lastBest = bestIdx
	} else {
		switch {
		case pm.peerList[bestIdx].Loss() < pm.peerList[pm.lastBest].Loss():
			// lower loss is a must
			pm.lastBest = bestIdx
		case pm.peerList[bestIdx].Latency() < pm.peerList[pm.lastBest].Latency()*betterCoeficient:
			// reduce too much reroutes and move to other route only if it is xx% better
			pm.lastBest = bestIdx
		}

	}

	return pm.peerList[pm.lastBest].gateway
}
