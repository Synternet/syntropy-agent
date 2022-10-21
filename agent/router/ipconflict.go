package router

import (
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

func (r *Router) hasIpConflict(addr netip.Prefix, groupID int) bool {
	for gid, routesGroup := range r.routes {
		if routesGroup.peerMonitor.HasNode(addr) ||
			routesGroup.serviceMonitor.HasAddress(addr) {
			if groupID != gid {
				logger.Error().Println(pkgName, addr.String(), "IP conflict. Connection GIDs:", groupID, gid)
				return true
			}
		}
	}

	return false
}

func (r *Router) resolveIpConflict() (count int) {
	for _, routeGroup := range r.routes {
		count += routeGroup.peerMonitor.ResolveIpConflict(r.hasIpConflict)
		count += routeGroup.serviceMonitor.ResolveIpConflict(r.hasIpConflict)
	}
	return count
}
