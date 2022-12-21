/**
 * Route selection algorithm, oriented at speed (lowest latency)
 **/
package speed

import (
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/peerlist"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/routeselector"
)

type speedRouteSelector struct {
	config    *routeselector.RouteSelectorConfig
	peerlist  *peerlist.PeerList
	bestRoute netip.Prefix
}

func New(peerlist *peerlist.PeerList, cfg *routeselector.RouteSelectorConfig) routeselector.PathSelector {
	return &speedRouteSelector{
		peerlist: peerlist,
		config:   cfg,
	}
}

func (srs *speedRouteSelector) BestPath() *routeselector.SelectedRoute {
	route := &routeselector.SelectedRoute{
		ID: 0, // Ivalidate last ID
		// IP is empty value, so  IP.IsValid()==false means delete route
		// Reason will be set bellow
	}
	if srs.peerlist.Count() == 0 {
		srs.bestRoute = netip.Prefix{}
		route.Reason = routeselector.NewReason(routeselector.ReasonRouteDelete, 0, 0)
		return route
	}

	route.Reason = srs.calculate()

	if srs.bestRoute.IsValid() {
		peer, ok := srs.peerlist.GetPeer(srs.bestRoute)

		if ok {
			if srs.config.RouteDeleteLossThreshold > 0 &&
				peer.Loss()*100 >= srs.config.RouteDeleteLossThreshold {
				route.Reason = routeselector.NewReason(routeselector.ReasonRouteDelete, 0, 0)
				return route
			}

			route.IP = srs.bestRoute.Addr()
			route.ID = peer.ConnectionID
		}
	}

	return route
}

func (srs *speedRouteSelector) calculate() *routeselector.RouteChangeReason {
	newIp := srs.peerlist.BestRoute()

	newStats, ok := srs.peerlist.GetPeer(newIp)
	// Did not found new best route (can this ever happen?)
	if !newIp.IsValid() || !ok {
		return routeselector.NewReason(routeselector.ReasonNoChange, 0, 0)
	}

	// No previous best route yet - choose the best
	prevStats, ok := srs.peerlist.GetPeer(srs.bestRoute)
	if !srs.bestRoute.IsValid() || !ok {
		srs.bestRoute = newIp
		return routeselector.NewReason(routeselector.ReasonNewRoute, 0, 0)
	}

	// lower loss is a must
	if newStats.Loss() < prevStats.Loss() {
		srs.bestRoute = newIp
		return routeselector.NewReason(routeselector.ReasonLoss,
			prevStats.Loss(), newStats.Loss())
	}

	// cannot compare latencies, if one does not have full statistics yet
	if newStats.StatsIncomplete() {
		return routeselector.NewReason(routeselector.ReasonNoChange, 0, 0)
	}

	// apply thresholds
	if prevStats.Latency()/newStats.Latency() >= srs.config.RerouteRatio &&
		prevStats.Latency()-newStats.Latency() >= srs.config.RerouteDiff {
		srs.bestRoute = newIp
		return routeselector.NewReason(routeselector.ReasonLatency,
			prevStats.Latency(), newStats.Latency())
	}

	// No changes - stay with old value
	return routeselector.NewReason(routeselector.ReasonNoChange, 0, 0)
}
