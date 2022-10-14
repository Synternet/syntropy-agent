package peermon

import "net/netip"

// bestRouteIndex searches and returns current best route
// (using moving average)
func (pm *PeerMonitor) bestRoute() netip.Prefix {
	// find currently best route
	best := invalidBest()
	for ip := range pm.peerList {
		switch {
		// First valid entry found. Compare other against it
		// TODO: a little fishy here - I chose first entry as best path.
		// Reason for that - it was working same way when there was a slice instead of map here
		// And anyway this part will be rewritten very soon.
		// TODO: fix me asap.
		case !best.IsValid():
			best = ip
			// Ignore invalid ping results (have no results about it, cannot say anyting)
		case !pm.peerList[ip].Valid():
			continue
			// Best loss is always must
		case pm.peerList[ip].Loss() > pm.peerList[best].Loss():
			continue
		case pm.peerList[ip].Loss() < pm.peerList[best].Loss():
			best = ip
		// compare peer with lower latency
		case pm.peerList[ip].Latency() > 0 &&
			pm.peerList[ip].Latency() < pm.peerList[best].Latency():
			best = ip
		}
	}

	return best
}

func (pm *PeerMonitor) isLastBestValid() bool {
	return pm.lastBest != invalidBest() && pm.peerList[pm.lastBest] != nil
}

// This function compares currently active best with newly calculated best route
// And compares if route should be changed taken into account route change thresholds
// It changes PeerMonitor's lastBest and changeReason members
func bestPathLowestLatency(pm *PeerMonitor) (addr netip.Prefix, reason *RouteChangeReason) {
	newBest := pm.bestRoute()

	// Did not found new best route (can this ever happen?)
	if !newBest.IsValid() {
		return pm.lastBest, NewReason(reasonNoChange, 0, 0)
	}

	// No previous best route yet - choose the best
	if !pm.isLastBestValid() {
		return newBest, NewReason(reasonNewRoute, 0, 0)
	}

	// lower loss is a must
	if pm.peerList[newBest].Loss() < pm.peerList[pm.lastBest].Loss() {
		return newBest, NewReason(reasonLoss,
			pm.peerList[pm.lastBest].Loss(),
			pm.peerList[newBest].Loss())
	}

	// cannot compare latencies, if one does not have full statistics yet
	if pm.peerList[newBest].StatsIncomplete() {
		return pm.lastBest, NewReason(reasonNoChange, 0, 0)
	}

	// apply thresholds
	if pm.peerList[pm.lastBest].Latency()/pm.peerList[newBest].Latency() >= pm.config.RerouteRatio &&
		pm.peerList[pm.lastBest].Latency()-pm.peerList[newBest].Latency() >= pm.config.RerouteDiff {
		return newBest, NewReason(reasonLatency,
			pm.peerList[pm.lastBest].Latency(),
			pm.peerList[newBest].Latency())
	}

	return pm.lastBest, NewReason(reasonNoChange, 0, 0)
}

func bestPathPreferPublic(pm *PeerMonitor) (addr netip.Prefix, reason *RouteChangeReason) {
	var publicIP netip.Prefix
	for ip, e := range pm.peerList {
		if e.IsPublic() {
			publicIP = ip
			break
		}
	}

	newBest := pm.bestRoute()
	if newBest == publicIP {
		if pm.lastBest == publicIP {
			return newBest, NewReason(reasonNoChange, 0, 0)
		} else {
			return newBest, NewReason(reasonLatency, 0, 0)
		}
	}

	// Did not found new best route (can this ever happen?)
	if !newBest.IsValid() {
		return pm.lastBest, NewReason(reasonNoChange, 0, 0)
	}

	// lower loss is a must
	if pm.peerList[newBest].Loss() < pm.peerList[publicIP].Loss() {
		return newBest, NewReason(reasonLoss, 0, 0)
	}

	// best route still does not completed full stats cycle
	if pm.peerList[newBest].StatsIncomplete() {
		if pm.isLastBestValid() {
			// Use last best, if it was selected alredy
			return pm.lastBest, NewReason(reasonNoChange, 0, 0)
		} else if pm.peerList[publicIP].Loss() == 0 && pm.peerList[publicIP].Latency() > 0 {
			// Fallback to public, if it is usable
			return publicIP, NewReason(reasonNewRoute, 0, 0)
		}
	}

	// apply thresholds
	if pm.peerList[publicIP].Latency()/pm.peerList[newBest].Latency() >= pm.config.RerouteRatio &&
		pm.peerList[publicIP].Latency()-pm.peerList[newBest].Latency() >= pm.config.RerouteDiff {
		return newBest, NewReason(reasonLatency,
			pm.peerList[publicIP].Latency(),
			pm.peerList[newBest].Latency())
	}

	return publicIP, NewReason(reasonLatency, 0, 0)
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
	route := &SelectedRoute{
		ID: 0, // Ivalidate last ID
		// IP is empty value, so  IP.IsValid()==false means delete route
		// Reason will be set bellow
	}

	if len(pm.peerList) == 0 {
		pm.lastBest = invalidBest()
		route.Reason = NewReason(reasonRouteDelete, 0, 0)
		return route
	}

	pm.lastBest, route.Reason = pm.pathSelector(pm)

	if pm.lastBest.IsValid() {

		if pm.config.RouteDeleteLossThreshold > 0 && pm.peerList[pm.lastBest].Loss()*100 >= pm.config.RouteDeleteLossThreshold {
			route.Reason = NewReason(reasonRouteDelete, 0, 0)
			return route
		}

		route.IP = pm.lastBest.Addr()
		route.ID = pm.peerList[pm.lastBest].connectionID
	}

	return route
}
