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
)

type directRouteSelector struct {
	config    *routeselector.RouteSelectorConfig
	peerlist  *peerlist.PeerList
	bestRoute netip.Prefix
}

func New(peerlist *peerlist.PeerList, cfg *routeselector.RouteSelectorConfig) routeselector.PathSelector {
	return &directRouteSelector{
		peerlist: peerlist,
		config:   cfg,
	}
}

func (drs *directRouteSelector) BestPath() *routeselector.SelectedRoute {
	route := &routeselector.SelectedRoute{
		ID: 0, // Ivalidate last ID
		// IP is empty value, so  IP.IsValid()==false means delete route
		// Reason will be set bellow
	}
	if drs.peerlist.Count() == 0 {
		drs.bestRoute = netip.Prefix{}
		route.Reason = routeselector.NewReason(routeselector.ReasonRouteDelete, 0, 0)
		return route
	}

	route.Reason = drs.calculate()

	if drs.bestRoute.IsValid() {
		peer, ok := drs.peerlist.GetPeer(drs.bestRoute)

		if ok {
			if drs.config.RouteDeleteLossThreshold > 0 &&
				peer.Loss()*100 >= drs.config.RouteDeleteLossThreshold {
				route.Reason = routeselector.NewReason(routeselector.ReasonRouteDelete, 0, 0)
				return route
			}

			route.IP = drs.bestRoute.Addr()
			route.ID = peer.ConnectionID
		}
	}

	return route
}

func (drs *directRouteSelector) calculate() *routeselector.RouteChangeReason {
	newIp := drs.peerlist.BestRoute()

	newStats, ok := drs.peerlist.GetPeer(newIp)
	// Did not found new best route (can this ever happen?)
	// Change nothing and leave as it is
	if !newIp.IsValid() || !ok {
		return routeselector.NewReason(routeselector.ReasonNoChange, 0, 0)
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
		drs.bestRoute = newIp
		return routeselector.NewReason(routeselector.ReasonNewRoute, 0, 0)
	}

	// lower loss is a must
	if newStats.Loss() < publicStats.Loss() {
		drs.bestRoute = newIp
		return routeselector.NewReason(routeselector.ReasonLoss, 0, 0)
	}

	// best route still does not completed full stats cycle
	if newStats.StatsIncomplete() {
		if drs.bestRoute.IsValid() {
			// Use last best, if it was selected alredy
			return routeselector.NewReason(routeselector.ReasonNoChange, 0, 0)
		} else if publicStats.Loss() == 0 && publicStats.Latency() > 0 {
			// Fallback to public, if it is usable
			drs.bestRoute = publicIp
			return routeselector.NewReason(routeselector.ReasonNewRoute, 0, 0)
		}
	}

	var reason *routeselector.RouteChangeReason
	prevStats, prevStatsOK := drs.peerlist.GetPeer(drs.bestRoute)

	switch {
	// if best route is public, and we prefer public - change to it immediately
	case newIp == publicIp:
		drs.bestRoute = publicIp
		if prevStatsOK {
			reason = routeselector.NewReason(routeselector.ReasonLatency,
				prevStats.Latency(), newStats.Latency())
		} else {
			reason = routeselector.NewReason(routeselector.ReasonNewRoute, 0, 0)
		}

	// Compare against public route
	case publicStats.Latency()/newStats.Latency() >= drs.config.RerouteRatio &&
		publicStats.Latency()-newStats.Latency() >= drs.config.RerouteDiff:
		drs.bestRoute = newIp
		reason = routeselector.NewReason(routeselector.ReasonLatency,
			publicStats.Latency(), newStats.Latency())

	// try prevent instant route flopping, if latencies are close to "the red line"
	// NB: in this case use half of configured thresholds.
	// Maybe we need another configuration variable for this?
	case prevStatsOK &&
		publicStats.Latency()/prevStats.Latency() >= drs.config.RerouteRatio/2 &&
		publicStats.Latency()-prevStats.Latency() >= drs.config.RerouteDiff/2:
		reason = routeselector.NewReason(routeselector.ReasonNoChange, 0, 0)

	// No big improvements can be gained (with applied thresholds)
	// Stick with direct/public route
	default:
		drs.bestRoute = publicIp
		switch {
		case drs.bestRoute == publicIp:
			reason = routeselector.NewReason(routeselector.ReasonNoChange, 0, 0)
		case prevStatsOK:
			reason = routeselector.NewReason(routeselector.ReasonLatency,
				prevStats.Latency(), publicStats.Latency())
		default:
			reason = routeselector.NewReason(routeselector.ReasonNewRoute, 0, publicStats.Latency())

		}
	}

	return reason
}
