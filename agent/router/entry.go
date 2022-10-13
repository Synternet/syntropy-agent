package router

import (
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon"
	"github.com/SyntropyNet/syntropy-agent/agent/router/servicemon"
)

type routerGroupEntry struct {
	peerMonitor    *peermon.PeerMonitor
	serviceMonitor *servicemon.ServiceMonitor
}

func (r *Router) findOrCreate(groupID int) *routerGroupEntry {
	routesGroup, ok := r.routes[groupID]
	if !ok {
		routesGroup = new(routerGroupEntry)
		routesGroup.peerMonitor = peermon.New(&r.pmCfg, groupID)
		routesGroup.serviceMonitor = servicemon.New(routesGroup.peerMonitor, groupID)
		r.routes[groupID] = routesGroup
	}
	return routesGroup
}

func (r *Router) find(groupID int) (*routerGroupEntry, bool) {
	routesGroup, ok := r.routes[groupID]
	return routesGroup, ok
}
