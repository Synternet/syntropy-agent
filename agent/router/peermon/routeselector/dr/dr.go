/**
 * Route selection algorithm, oriented at reducing costs
 * and should return to direct route (public route)
 * when it becomes acceptable.
 **/
package dr

import (
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/peerlist"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/routeselector"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const pkgName = "DirectRouteSelector"

type candidate struct {
	route netip.Prefix
	count uint
}

func (c *candidate) reset(ip netip.Prefix) {
	c.route = ip
	c.count = 0
}

type directRouteSelector struct {
	config    *routeselector.RouteSelectorConfig
	peerlist  *peerlist.PeerList
	bestRoute netip.Prefix
	reason    routeselector.RouteChangeReason
	underdog  candidate
}

func New(peerlist *peerlist.PeerList, cfg *routeselector.RouteSelectorConfig) routeselector.PathSelector {
	return &directRouteSelector{
		peerlist: peerlist,
		config:   cfg,
	}
}

func (drs *directRouteSelector) BestPath() *routeselector.SelectedRoute {
	route := &routeselector.SelectedRoute{
		ID: 0, // Invalidate last ID
		// IP is empty value, so  IP.IsValid()==false means delete route
		Reason: &drs.reason,
	}
	if drs.peerlist.Count() == 0 {
		drs.bestRoute = netip.Prefix{}
		drs.reason.Set(routeselector.ReasonRouteDelete, 0, 0)
		return route
	}

	drs.calculate()

	if drs.bestRoute.IsValid() {
		peer, ok := drs.peerlist.GetPeer(drs.bestRoute)

		if ok {
			if drs.config.RouteDeleteLossThreshold > 0 &&
				peer.Loss()*100 >= drs.config.RouteDeleteLossThreshold {
				route.Reason.Set(routeselector.ReasonRouteDelete, 0, 0)
				return route
			}

			route.IP = drs.bestRoute.Addr()
			route.ID = peer.ConnectionID
			// Reason is already assigned by pointer
		}
	}

	return route
}

func (drs *directRouteSelector) calculate() {
	/**
	 * Errors and unexpected situations handling
	**/

	newIp := drs.peerlist.BestRoute()
	newStats, ok := drs.peerlist.GetPeer(newIp)
	// Did not found new best route (can this ever happen?)
	// Change nothing and leave as it is
	if !newIp.IsValid() || !ok {
		logger.Warning().Println(pkgName, "Peer", newIp.String(), "not found")
		drs.reason.Set(routeselector.ReasonNoChange, 0, 0)
		return
	}

	var publicIp netip.Prefix
	drs.peerlist.Iterate(func(ip netip.Prefix, peer *peerlist.PeerInfo) {
		if peer.IsPublic() {
			publicIp = ip
		}
	})
	publicStats, ok := drs.peerlist.GetPeer(publicIp)
	// Did not found public route - (can this ever happen?)
	// Use calculated best route
	if !publicIp.IsValid() || !ok {
		logger.Warning().Println(pkgName, "Public route not found")
		drs.bestRoute = newIp
		drs.reason.Set(routeselector.ReasonNewRoute, 0, newStats.Latency())
		return
	}

	// Previous best statistics, needed for calculation
	var prevStatsLatency float32
	var prevStatsLoss float32
	prevStats, prevStatsOK := drs.peerlist.GetPeer(drs.bestRoute)
	if prevStatsOK {
		prevStatsLatency = prevStats.Latency()
		prevStatsLoss = prevStats.Loss()
	}

	// best route still does not completed full stats cycle
	if newStats.StatsIncomplete() {
		if drs.bestRoute.IsValid() {
			// Use last best, if it was selected alredy
			drs.reason.Set(routeselector.ReasonNoChange, prevStatsLatency, prevStatsLatency)
		} else if publicStats.Loss() == 0 && publicStats.Latency() > 0 {
			// Fallback to public, if it is usable
			drs.bestRoute = publicIp
			drs.reason.Set(routeselector.ReasonNewRoute, 0, publicStats.Latency())
		}
		return
	}

	/**
	 * Now lets proceed to the complicated best route choice :)
	**/

	// if best route is public, and we prefer public - change to it immediately
	if newIp == publicIp {
		if newIp == drs.bestRoute {
			// No change is needed if best route is current route
			drs.reason.Set(routeselector.ReasonNoChange, prevStatsLatency, newStats.Latency())
			return
		}
		drs.bestRoute = publicIp
		drs.reason.Set(routeselector.ReasonLatency, prevStatsLatency, newStats.Latency())
		drs.underdog.reset(drs.bestRoute)
		return
	}

	// temp variables to store intermediate calculation best values
	newBestRoute := newIp
	newBestStats := newStats

	// lower loss is a must
	if (publicStats.Loss() + prevStatsLoss) > 0 {
		if publicStats.Loss() < newBestStats.Loss() {
			newBestRoute = publicIp
			newBestStats = publicStats
		}
		if prevStatsOK && prevStatsLoss < newBestStats.Loss() {
			newBestRoute = drs.bestRoute
			newBestStats = prevStats
		}
		if newBestRoute == drs.bestRoute {
			drs.reason.Set(routeselector.ReasonNoChange, prevStatsLatency, newBestStats.Latency())
		} else {
			drs.bestRoute = newBestRoute
			if prevStatsOK {
				drs.reason.Set(routeselector.ReasonLoss, prevStatsLoss, newBestStats.Loss())
			} else {
				drs.reason.Set(routeselector.ReasonNewRoute, 0, newBestStats.Loss())
			}
		}
		return
	}

	// Compare against public route
	if publicStats.Latency()/newStats.Latency() >= drs.config.RerouteRatio &&
		publicStats.Latency()-newStats.Latency() >= drs.config.RerouteDiff {
		if !prevStatsOK {
			// no prev stats - cannot compare, so change to the best
			newBestRoute = newIp
		} else if drs.bestRoute == newIp {
			// if new best is current = no change is needed
			newBestRoute = drs.bestRoute
		} else if prevStats.Latency()/newStats.Latency() >= drs.config.RerouteRatio/2 &&
			prevStats.Latency()-newStats.Latency() >= drs.config.RerouteDiff/2 {
			// try prevent instant route flopping, if latencies are close to "the red line"
			// NB: in this case use half of configured thresholds.
			// Maybe we need another configuration variable for this?
			// NOTE: New moment best is always not worse (better or equal) than current,
			// so no need to compare vise versa
			newBestRoute = newIp
		} else {
			// new best route is not that best. Stay on current route
			// But keep an eye - alternative may be better in long term
			newBestRoute = drs.bestRoute
			// TODO: newIp  underdog++
		}
	} else {
		if drs.bestRoute == publicIp {
			// if new best is not better than current public = no change is needed
			newBestRoute = drs.bestRoute
		} else if !prevStatsOK {
			// no prev stats - cannot compare, so change to the best
			newBestRoute = publicIp
		} else if prevStats.Latency() > publicStats.Latency() {
			newBestRoute = publicIp
		} else if publicStats.Latency()/prevStats.Latency() >= drs.config.RerouteRatio/2 &&
			publicStats.Latency()-prevStats.Latency() >= drs.config.RerouteDiff/2 {
			// try prevent instant route flopping, if latencies are close to "the red line"
			// NB: in this case use half of configured thresholds.
			// Maybe we need another configuration variable for this?
			newBestRoute = drs.bestRoute
			// TODO: public underdog++
		} else {
			newBestRoute = publicIp
		}
	}

	if newBestRoute == drs.bestRoute {
		drs.reason.Set(routeselector.ReasonNoChange, prevStatsLatency, newBestStats.Latency())
	} else {
		drs.bestRoute = newBestRoute
		if prevStatsOK {
			drs.reason.Set(routeselector.ReasonLatency, prevStatsLatency, newBestStats.Latency())
		} else {
			drs.reason.Set(routeselector.ReasonNewRoute, 0, newBestStats.Loss())
		}
	}
}
