package peermon

import "github.com/SyntropyNet/syntropy-agent/internal/config"

// bestRouteIndex searches and returns current best route
// (using moving average)
func (pm *PeerMonitor) bestRouteIndex() int {
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

	return bestIdx
}

// This function compares currently active best with newly calculated best route
// And compares if route should be changed taken into account route change thresholds
// It changes PeerMonitor's lastBest and changeReason members
func bestPathLowestLatency(pm *PeerMonitor) (index, reason int) {
	newIdx := pm.bestRouteIndex()

	// No previous best route yet - choose the best
	if pm.lastBest == invalidBestIndex || pm.lastBest >= len(pm.peerList) {
		return newIdx, reasonNewRoute
	}

	// lower loss is a must
	if pm.peerList[newIdx].Loss() < pm.peerList[pm.lastBest].Loss() {
		return newIdx, reasonLoss
	}

	// cannot compare latencies, if one does not have full statistics yet
	if pm.peerList[newIdx].StatsIncomplete() {
		return pm.lastBest, reasonNoChange
	}

	// apply thresholds
	diff, ratio := config.RerouteThresholds()
	if pm.peerList[pm.lastBest].Latency()/pm.peerList[newIdx].Latency() >= ratio &&
		pm.peerList[pm.lastBest].Latency()-pm.peerList[newIdx].Latency() >= diff {
		return newIdx, reasonLatency
	}

	return pm.lastBest, reasonNoChange
}

func bestPathPreferPublic(pm *PeerMonitor) (index, reason int) {
	var publicIdx int
	for i := 0; i < len(pm.peerList); i++ {
		if pm.peerList[i].IsPublic() {
			publicIdx = i
			break
		}
	}

	newIdx := pm.bestRouteIndex()

	// No previous best route yet
	// (or the best route still does not completed full stats cycle)
	// try choosing public (only if it is usable)
	if pm.lastBest == invalidBestIndex ||
		pm.lastBest >= len(pm.peerList) ||
		pm.peerList[newIdx].StatsIncomplete() {
		if pm.peerList[publicIdx].Loss() == 0 && pm.peerList[publicIdx].Latency() > 0 {
			return publicIdx, reasonNewRoute
		}
	}

	// lower loss is a must
	if pm.peerList[newIdx].Loss() < pm.peerList[publicIdx].Loss() {
		return newIdx, reasonLoss
	}

	// apply thresholds
	diff, ratio := config.RerouteThresholds()
	if pm.peerList[publicIdx].Latency()/pm.peerList[newIdx].Latency() >= ratio &&
		pm.peerList[publicIdx].Latency()-pm.peerList[newIdx].Latency() >= diff {
		return newIdx, reasonLatency
	}

	return pm.lastBest, reasonNoChange
}

// BestPath returns best route gateway.
// Best route is:
//  * Lowest packet loss
//  * possible lowest latency
// But in order for not to fluctuate between 2 routes, when latency is the same
// so once one best route is found - do not switch to another route, unless it is (betterPercent)% better
func (pm *PeerMonitor) BestPath() *SelectedRoute {
	pm.RLock()
	defer pm.RUnlock()

	if len(pm.peerList) == 0 {
		pm.lastBest = invalidBestIndex
		pm.changeReason = reasonNewRoute
		return nil
	}

	pm.lastBest, pm.changeReason = pm.pathSelector(pm)

	lossThreshold := config.GetRouteDeleteThreshold()
	if lossThreshold > 0 && pm.peerList[pm.lastBest].Loss()*100 >= float32(lossThreshold) {
		return nil
	}

	return &SelectedRoute{
		IP: pm.peerList[pm.lastBest].ip,
		ID: pm.peerList[pm.lastBest].connectionID,
	}
}
