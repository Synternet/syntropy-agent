package peerlist

import "net/netip"

// Best route index is not set yet
func invalidBest() netip.Prefix {
	return netip.Prefix{}
}

// BestRoute searches and returns current momentary best route
// Momentary as stated above isn't trully momentary - it uses moving average
// See config keys SYNTROPY_PEERCHECK_TIME and SYNTROPY_PEERCHECK_WINDOW
// Default is 2 minutes moving average
func (pl *PeerList) BestRoute() netip.Prefix {
	// find currently best route
	best := invalidBest()
	for ip := range pl.peers {
		switch {
		// First valid entry found. Compare other against it
		case !best.IsValid():
			best = ip
			// Ignore invalid ping results (have no results about it, cannot say anyting)
		case !pl.peers[ip].Valid():
			continue
			// Best loss is always must
		case pl.peers[ip].Loss() > pl.peers[best].Loss():
			continue
		case pl.peers[ip].Loss() < pl.peers[best].Loss():
			best = ip
		// compare peer with lower latency
		case pl.peers[ip].Latency() > 0 &&
			pl.peers[ip].Latency() < pl.peers[best].Latency():
			best = ip
		}
	}

	return best
}
