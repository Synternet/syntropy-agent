package router

import (
	"errors"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent-go/agent/router/servicemon"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/netcfg"
)

func (r *Router) ServiceAdd(netpath *common.SdnNetworkPath, destination string) (*common.RouteResult, *peeradata.PeerActiveDataEntry) {
	routeRes := common.RouteResult{
		IP: destination,
	}

	if err := r.serviceMonitor.Add(netpath, destination); err != nil {
		if errors.Is(err, servicemon.ErrSdnRouteExists) {
			logger.Debug().Println(pkgName, "skip existing SDN route to", destination)
		} else {
			routeRes.Error = err
		}
		return nil, nil
	}

	logger.Info().Println(pkgName, "Route add ", destination, " via ", netpath.Gateway)
	routeRes.Error = netcfg.RouteAdd(netpath.Ifname, netpath.Gateway, destination)
	if routeRes.Error != nil {
		logger.Error().Println(pkgName, "route add error:", routeRes.Error)
		return &routeRes, nil
	}

	return &routeRes,
		&peeradata.PeerActiveDataEntry{
			PreviousConnID: 0,
			ConnectionID:   netpath.ConnectionID,
			GroupID:        netpath.GroupID,
			Timestamp:      time.Now().Format(env.TimeFormat),
		}
}

func (r *Router) ServiceDel(netpath *common.SdnNetworkPath, destination string) *common.RouteResult {
	routeRes := common.RouteResult{
		IP: destination,
	}

	r.serviceMonitor.Del(netpath, destination)

	logger.Info().Println(pkgName, "Route delete ", destination, " via ", netpath.Gateway)

	routeRes.Error = netcfg.RouteDel(netpath.Ifname, destination)
	if routeRes.Error != nil {
		logger.Error().Println(pkgName, destination, "route delete error", routeRes.Error)
	}

	return &routeRes
}
