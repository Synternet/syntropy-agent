package router

import (
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/netcfg"
)

func (r *Router) PingProcess(pr []multiping.PingResult) {
	r.peerMonitor.PingProcess(pr)
}

func (r *Router) PeerAdd(netpath *common.SdnNetworkPath, destination string) common.RouteResult {
	entry := common.RouteResult{
		IP: destination,
	}

	// peerMonitor needs only IP, not full CIDR
	parts := strings.Split(destination, "/")
	r.peerMonitor.AddNode(netpath.Gateway, parts[0])

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

	// peerMonitor needs only IP, not full CIDR
	parts := strings.Split(destination, "/")
	r.peerMonitor.DelNode(parts[0])

	entry.Error = netcfg.RouteDel(netpath.Ifname, destination)
	if entry.Error != nil {
		logger.Error().Println(pkgName, destination, "route delete error", entry.Error)
	}

	return entry
}
