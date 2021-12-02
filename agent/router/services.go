package router

import (
	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

func (r *Router) ServiceAdd(netpath *common.SdnNetworkPath, destination string) error {
	routesGroup := r.findOrCreate(netpath.GroupID)

	return routesGroup.serviceMonitor.Add(netpath, destination)
}

func (r *Router) ServiceDel(netpath *common.SdnNetworkPath, destination string) error {
	routesGroup, ok := r.find(netpath.GroupID)
	if !ok {
		// Was asked to delete non-existing service route.
		// So its like I've done what I was asked - do not disturb caller
		logger.Warning().Printf("%s delete service route to %s: route group %d does not exist\n",
			pkgName, destination, netpath.GroupID)
		return nil
	}

	return routesGroup.serviceMonitor.Del(netpath, destination)
}

func (r *Router) serviceApply() ([]*routestatus.Connection, []*peeradata.Entry) {
	routeStatusCons := []*routestatus.Connection{}
	peersActiveData := []*peeradata.Entry{}
	for _, route := range r.routes {
		rsc, pad := route.serviceMonitor.Apply()
		routeStatusCons = append(routeStatusCons, rsc...)
		peersActiveData = append(peersActiveData, pad...)
	}

	return routeStatusCons, peersActiveData
}
