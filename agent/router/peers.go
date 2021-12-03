package router

import (
	"fmt"
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/netcfg"
)

func (r *Router) PeerAdd(netpath *common.SdnNetworkPath, destination string) common.RouteResult {
	entry := common.RouteResult{
		IP: destination,
	}

	routesGroup := r.findOrCreate(netpath.GroupID)
	pm := routesGroup.peerMonitor

	// peerMonitor needs only IP, not full CIDR
	parts := strings.Split(destination, "/")
	pm.AddNode(netpath.Gateway, parts[0])

	routeConflict, conflictIfName := netcfg.RouteConflict(destination)
	if routeConflict {
		// Route already exists. So we have 2 options here
		if strings.HasPrefix(conflictIfName, env.InterfaceNamePrefix) {
			// If route is via other Syntropy interface - change the route to this interface
			// TODO: in future optimise this, and if existing link is legal - try not to change it
			logger.Info().Println(pkgName, "Route update ", destination, " via ", netpath.Gateway, "/", netpath.Ifname)
			entry.Error = netcfg.RouteReplace(netpath.Ifname, "", destination)
			if entry.Error != nil {
				logger.Error().Println(pkgName, "route update error:", entry.Error)
				return entry
			}
		} else {
			// If route is not via SYNTROPY - inform error
			entry.Error = fmt.Errorf("route to %s conflict: wanted via %s exists on %s",
				destination, netpath.Gateway, conflictIfName)
			logger.Error().Println(pkgName, "route add error:", entry.Error)
			return entry
		}

	} else {
		// clean case - no route conflict. Simply add the route
		logger.Info().Println(pkgName, "Route add ", destination, " via ", netpath.Gateway, "/", netpath.Ifname)
		entry.Error = netcfg.RouteAdd(netpath.Ifname, "", destination)
		if entry.Error != nil {
			logger.Error().Println(pkgName, "route add error:", entry.Error)
			return entry
		}
	}

	return entry
}

func (r *Router) PeerDel(netpath *common.SdnNetworkPath, destination string) common.RouteResult {
	entry := common.RouteResult{
		IP: destination,
	}
	routesGroup, ok := r.find(netpath.GroupID)
	if !ok {
		// Was asked to delete non-existing route.
		// So its like I've done what I was asked - do not disturb caller
		logger.Warning().Printf("%s delete peer route to %s: route group %d does not exist\n",
			pkgName, destination, netpath.GroupID)
		return entry
	}
	pm := routesGroup.peerMonitor

	// peerMonitor needs only IP, not full CIDR
	parts := strings.Split(destination, "/")
	pm.DelNode(parts[0])

	entry.Error = netcfg.RouteDel(netpath.Ifname, destination)
	if entry.Error != nil {
		logger.Error().Println(pkgName, destination, "route delete error", entry.Error)
	}

	return entry
}

func (r *Router) PingProcess(pr *multiping.PingData) {
	for _, pm := range r.routes {
		pm.peerMonitor.PingProcess(pr)
	}
}
