package peermon

// bestRouteIndex searches and returns current best route
// (using moving average)
func (pm *PeerMonitor) bestRouteIndex() int {
	// find currently best route
	bestIdx := 0
	for i := bestIdx + 1; i < len(pm.peerList); i++ {
		switch {
		// Ignore invalid ping results (have no results about it, cannot say anyting)
		case !pm.peerList[i].Valid():
			continue
		// Best loss is always must
		case pm.peerList[i].Loss() > pm.peerList[bestIdx].Loss():
			continue
		case pm.peerList[i].Loss() < pm.peerList[bestIdx].Loss():
			bestIdx = i
		// compare peer with lower latency
		case pm.peerList[i].Latency() > 0 &&
			pm.peerList[i].Latency() < pm.peerList[bestIdx].Latency():
			bestIdx = i
		}
	}

	return bestIdx
}

func (pm *PeerMonitor) isLastBestValid() bool {
	return pm.lastBest != invalidBestIndex && pm.lastBest < len(pm.peerList)
}

// This function compares currently active best with newly calculated best route
// And compares if route should be changed taken into account route change thresholds
// It changes PeerMonitor's lastBest and changeReason members
func bestPathLowestLatency(pm *PeerMonitor) (index, reason int) {
	newIdx := pm.bestRouteIndex()

	// No previous best route yet - choose the best
	if !pm.isLastBestValid() {
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
	if pm.peerList[pm.lastBest].Latency()/pm.peerList[newIdx].Latency() >= pm.config.RerouteRatio &&
		pm.peerList[pm.lastBest].Latency()-pm.peerList[newIdx].Latency() >= pm.config.RerouteDiff {
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

	if newIdx == publicIdx {
		if pm.lastBest == publicIdx {
			return newIdx, reasonNoChange
		} else {
			return newIdx, reasonLatency
		}
	}

	// Did not found new best route (can this ever happen?)
	if newIdx == invalidBestIndex {
		return pm.lastBest, reasonNoChange
	}

	// lower loss is a must
	if pm.peerList[newIdx].Loss() < pm.peerList[publicIdx].Loss() {
		return newIdx, reasonLoss
	}

	// best route still does not completed full stats cycle
	if pm.peerList[newIdx].StatsIncomplete() {
		if pm.isLastBestValid() {
			// Use last best, if it was selected alredy
			return pm.lastBest, reasonNoChange
		} else if pm.peerList[publicIdx].Loss() == 0 && pm.peerList[publicIdx].Latency() > 0 {
			// Fallback to public, if it is usable
			return publicIdx, reasonNewRoute
		}
	}

	// apply thresholds
	if pm.peerList[publicIdx].Latency()/pm.peerList[newIdx].Latency() >= pm.config.RerouteRatio &&
		pm.peerList[publicIdx].Latency()-pm.peerList[newIdx].Latency() >= pm.config.RerouteDiff {
		return newIdx, reasonLatency
	}

	return publicIdx, reasonLatency
}

// BestPath returns best route gateway.
// Best route is:
//   - Lowest packet loss
//   - possible lowest latency
//
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

	if pm.config.RouteDeleteLossThreshold > 0 && pm.peerList[pm.lastBest].Loss()*100 >= pm.config.RouteDeleteLossThreshold {
		return nil
	}

	return &SelectedRoute{
		IP: pm.peerList[pm.lastBest].ip,
		ID: pm.peerList[pm.lastBest].connectionID,
	}
}
