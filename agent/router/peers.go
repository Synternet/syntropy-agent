package router

import (
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
)

func (r *Router) peerAdd(netpath *common.SdnNetworkPath) error {
	dest := netip.PrefixFrom(netpath.Gateway, netpath.Gateway.BitLen()) // single address

	routesGroup := r.findOrCreate(netpath.GroupID)

	routesGroup.peerMonitor.AddNode(netpath.Ifname, netpath.PublicKey,
		dest, netpath.ConnectionID, r.hasIpConflict(dest, netpath.GroupID))

	return nil
}

func (r *Router) peerDel(netpath *common.SdnNetworkPath) error {
	dest := netip.PrefixFrom(netpath.Gateway, netpath.Gateway.BitLen()) // single address

	routesGroup, ok := r.find(netpath.GroupID)
	if !ok {
		// Was asked to delete non-existing route.
		// So its like I've done what I was asked - do not disturb caller
		logger.Warning().Printf("%s delete peer route to %s: route group %d does not exist\n",
			pkgName, netpath.Gateway, netpath.GroupID)
		return nil
	}

	routesGroup.peerMonitor.DelNode(dest)

	return nil
}

func (r *Router) PingProcess(pr *multiping.PingData) {
	r.Lock()
	defer r.Unlock()

	for _, pm := range r.routes {
		pm.peerMonitor.PingProcess(pr)
	}
	// After processing ping results check for a better route for services
	r.rerouteServices()
}
