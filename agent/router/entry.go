package router

import (
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon"
	"github.com/SyntropyNet/syntropy-agent/agent/router/servicemon"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
)

type routerGroupEntry struct {
	peerMonitor    *peermon.PeerMonitor
	serviceMonitor *servicemon.ServiceMonitor
}

func newRouteGroupEntry() *routerGroupEntry {
	rge := &routerGroupEntry{}
	rge.peerMonitor = peermon.New(config.PeerCheckWindow())
	rge.serviceMonitor = servicemon.New(rge.peerMonitor)
	return rge
}

func (r *Router) findOrCreate(groupID int) *routerGroupEntry {
	routesGroup, ok := r.routes[groupID]
	if !ok {
		routesGroup = newRouteGroupEntry()
		r.routes[groupID] = routesGroup
	}
	return routesGroup
}

func (r *Router) find(groupID int) (*routerGroupEntry, bool) {
	routesGroup, ok := r.routes[groupID]
	return routesGroup, ok
}
