package router

import (
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

func (r *Router) serviceAdd(netpath *common.SdnNetworkPath, destination netip.Prefix) error {
	isIPconflict := r.hasIpConflict(destination, netpath.GroupID)

	routesGroup := r.findOrCreate(netpath.GroupID)

	return routesGroup.serviceMonitor.Add(netpath, destination, isIPconflict)
}

func (r *Router) serviceDel(netpath *common.SdnNetworkPath, destination netip.Prefix) error {
	routesGroup, ok := r.find(netpath.GroupID)
	if !ok {
		// Was asked to delete non-existing service route.
		// So its like I've done what I was asked - do not disturb caller
		logger.Warning().Printf("%s delete service route to %s: route group %d does not exist\n",
			pkgName, destination, netpath.GroupID)
		return nil
	}
	logger.Debug().Println(pkgName, "Delete service route to", netpath.Gateway, "via", netpath.Gateway)

	return routesGroup.serviceMonitor.Del(netpath, destination)
}
