package router

import (
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
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

	entry.Error = netcfg.RouteAdd(netpath.Ifname, netpath.Gateway, destination)
	if entry.Error != nil {
		logger.Error().Println(pkgName, "route add error:", entry.Error)
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

func (r *Router) PingProcess(pr *multiping.PingResult) {
	for _, pm := range r.routes {
		pm.peerMonitor.PingProcess(pr)
	}
}
