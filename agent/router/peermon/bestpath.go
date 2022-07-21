package peermon

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

func (pm *PeerMonitor) isLastBestValid() bool {
	return pm.lastBest != invalidBestIndex && pm.lastBest < len(pm.peerList)
}

// This function compares currently active best with newly calculated best route
// And compares if route should be changed taken into account route change thresholds
// It changes PeerMonitor's lastBest and changeReason members
func bestPathLowestLatency(pm *PeerMonitor) (index int, reason *RouteChangeReason) {
	newIdx := pm.bestRouteIndex()

	// No previous best route yet - choose the best
	if !pm.isLastBestValid() {
		return newIdx, NewReason(reasonNewRoute, 0, 0)
	}

	// lower loss is a must
	if pm.peerList[newIdx].Loss() < pm.peerList[pm.lastBest].Loss() {
		return newIdx, NewReason(reasonLoss,
			pm.peerList[pm.lastBest].Loss(),
			pm.peerList[newIdx].Loss())
	}

	// cannot compare latencies, if one does not have full statistics yet
	if pm.peerList[newIdx].StatsIncomplete() {
		return pm.lastBest, NewReason(reasonNoChange, 0, 0)
	}

	// apply thresholds
	if pm.peerList[pm.lastBest].Latency()/pm.peerList[newIdx].Latency() >= pm.config.RerouteRatio &&
		pm.peerList[pm.lastBest].Latency()-pm.peerList[newIdx].Latency() >= pm.config.RerouteDiff {
		return newIdx, NewReason(reasonLatency,
			pm.peerList[pm.lastBest].Latency(),
			pm.peerList[newIdx].Latency())
	}

	return pm.lastBest, NewReason(reasonNoChange, 0, 0)
}

func bestPathPreferPublic(pm *PeerMonitor) (index int, reason *RouteChangeReason) {
	var publicIdx int
	for i := 0; i < len(pm.peerList); i++ {
		if pm.peerList[i].IsPublic() {
			publicIdx = i
			break
		}
	}

	newIdx := pm.bestRouteIndex()

	// best route still does not completed full stats cycle
	if pm.peerList[newIdx].StatsIncomplete() {
		if pm.isLastBestValid() {
			// Use last best, if it was selected alredy
			return pm.lastBest, NewReason(reasonNoChange, 0, 0)
		} else if pm.peerList[publicIdx].Loss() == 0 && pm.peerList[publicIdx].Latency() > 0 {
			// Fallback to public, if it is usable
			return publicIdx, NewReason(reasonNewRoute, 0, 0)
		}
	}

	// lower loss is a must
	if pm.peerList[newIdx].Loss() < pm.peerList[publicIdx].Loss() {
		return newIdx, NewReason(reasonLoss, 0, 0)
	}

	// apply thresholds
	if pm.peerList[publicIdx].Latency()/pm.peerList[newIdx].Latency() >= pm.config.RerouteRatio &&
		pm.peerList[publicIdx].Latency()-pm.peerList[newIdx].Latency() >= pm.config.RerouteDiff {
		return newIdx, NewReason(reasonLatency,
			pm.peerList[publicIdx].Latency(),
			pm.peerList[newIdx].Latency())
	}

	// No previous best route yet,
	// and best route `newIdx` is not better than public - fallback to public
	if !pm.isLastBestValid() {
		return publicIdx, NewReason(reasonNewRoute, 0, 0)
	}

	return pm.lastBest, NewReason(reasonNoChange, 0, 0)
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
	route := &SelectedRoute{
		ID: 0, // Ivalidate last ID
		// IP is empty value, so  IP.IsValid()==true means delete route
		// Reason will be set bellow
	}

	if len(pm.peerList) == 0 {
		pm.lastBest = invalidBestIndex
		route.Reason = NewReason(reasonRouteDelete, 0, 0)
		return route
	}

	pm.lastBest, route.Reason = pm.pathSelector(pm)

	if pm.config.RouteDeleteLossThreshold > 0 && pm.peerList[pm.lastBest].Loss()*100 >= pm.config.RouteDeleteLossThreshold {
		route.Reason = NewReason(reasonRouteDelete, 0, 0)
		return route
	}

	route.IP = pm.peerList[pm.lastBest].ip
	route.ID = pm.peerList[pm.lastBest].connectionID

	return route
}
