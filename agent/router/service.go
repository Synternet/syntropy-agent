package router

import (
	"errors"

	"github.com/SyntropyNet/syntropy-agent-go/agent/router/servicemon"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/generic/router"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/netcfg"
)

func (r *Router) ServiceAdd(netpath *router.SdnNetworkPath, destination string) router.RouteResult {
	entry := router.RouteResult{
		IP: destination,
	}

	if err := r.serviceMonitor.Add(netpath, destination); err != nil {
		if errors.Is(err, servicemon.ErrSdnRouteExists) {
			logger.Debug().Println(pkgName, "skip existing SDN route to", destination)
		} else {
			entry.Error = err
		}
		return entry
	}

	logger.Info().Println(pkgName, "Route add ", destination, " via ", netpath.Gateway)
	entry.Error = netcfg.RouteAdd(netpath.Ifname, netpath.Gateway, destination)
	if entry.Error != nil {
		logger.Error().Println(pkgName, "route add error:", entry.Error)
	}

	return entry
}

func (r *Router) ServiceDel(netpath *router.SdnNetworkPath, destination string) router.RouteResult {
	entry := router.RouteResult{
		IP: destination,
	}

	r.serviceMonitor.Del(netpath, destination)

	logger.Info().Println(pkgName, "Route delete ", destination, " via ", netpath.Gateway)

	entry.Error = netcfg.RouteDel(netpath.Ifname, destination)
	if entry.Error != nil {
		logger.Error().Println(pkgName, destination, "route delete error", entry.Error)
	}

	return entry
}
