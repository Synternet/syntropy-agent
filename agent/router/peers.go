package router

import (
	"strings"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

func (r *Router) PeerAdd(netpath *common.SdnNetworkPath, destination string) error {
	routesGroup := r.findOrCreate(netpath.GroupID)
	pm := routesGroup.peerMonitor

	// peerMonitor needs only IP, not full CIDR
	parts := strings.Split(destination, "/")
	pm.AddNode(netpath.Gateway, parts[0])

	err := netcfg.RouteAdd(netpath.Ifname, "", destination)
	if err != nil {
		logger.Error().Println(pkgName, "route add error:", err)
	}

	return err
}

func (r *Router) PeerDel(netpath *common.SdnNetworkPath, destination string) error {
	routesGroup, ok := r.find(netpath.GroupID)
	if !ok {
		// Was asked to delete non-existing route.
		// So its like I've done what I was asked - do not disturb caller
		logger.Warning().Printf("%s delete peer route to %s: route group %d does not exist\n",
			pkgName, destination, netpath.GroupID)
		return nil
	}
	pm := routesGroup.peerMonitor

	// peerMonitor needs only IP, not full CIDR
	parts := strings.Split(destination, "/")
	pm.DelNode(parts[0])

	err := netcfg.RouteDel(netpath.Ifname, destination)
	if err != nil {
		logger.Error().Println(pkgName, destination, "route delete error", err)
	}

	return err
}

func (r *Router) PingProcess(pr *multiping.PingData) {
	for _, pm := range r.routes {
		pm.peerMonitor.PingProcess(pr)
	}
}
