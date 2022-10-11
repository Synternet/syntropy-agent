package router

import (
	"fmt"
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

func (r *Router) PeerAdd(netpath *common.SdnNetworkPath) error {
	dest := netip.PrefixFrom(netpath.Gateway, netpath.Gateway.BitLen()) // single address

	if r.HasIpConflict(dest, netpath.GroupID) {
		return fmt.Errorf("%s duplicate IP address", dest.Addr().String())
	}

	routesGroup := r.findOrCreate(netpath.GroupID)

	routesGroup.peerMonitor.AddNode(netpath.Ifname, netpath.PublicKey,
		dest, netpath.ConnectionID)

	err := netcfg.RouteAdd(netpath.Ifname, nil, &dest)
	if err != nil {
		logger.Error().Println(pkgName, netpath.Gateway, "route add error:", err)
	}

	return err
}

func (r *Router) PeerDel(netpath *common.SdnNetworkPath) error {
	dest := netip.PrefixFrom(netpath.Gateway, netpath.Gateway.BitLen()) // single address

	routesGroup, ok := r.find(netpath.GroupID)
	if !ok {
		// Was asked to delete non-existing route.
		// So its like I've done what I was asked - do not disturb caller
		logger.Warning().Printf("%s delete peer route to %s: route group %d does not exist\n",
			pkgName, netpath.Gateway, netpath.GroupID)
		return nil
	}

	logger.Debug().Println(pkgName, "Delete peer route to", netpath.Gateway)
	routesGroup.peerMonitor.DelNode(dest)

	err := netcfg.RouteDel(netpath.Ifname, &dest)
	if err != nil {
		logger.Error().Println(pkgName, netpath.Gateway, "route delete error", err)
	}

	return err
}

func (r *Router) PingProcess(pr *multiping.PingData) {
	for _, pm := range r.routes {
		pm.peerMonitor.PingProcess(pr)
	}
	// After processing ping results check for a better route for services
	r.execute()
}

// Apply configured peers change
func (r *Router) peersApply() {
	// empty body now
}
