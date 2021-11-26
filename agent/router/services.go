package router

import (
	"errors"
	"fmt"
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent-go/agent/router/servicemon"
	"github.com/SyntropyNet/syntropy-agent-go/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/netcfg"
)

func (r *Router) ServiceAdd(netpath *common.SdnNetworkPath, destination string) (*routestatus.Entry, *peeradata.Entry) {
	var err error
	routesGroup := r.findOrCreate(netpath.GroupID)
	sm := routesGroup.serviceMonitor

	err = sm.Add(netpath, destination)
	if err != nil {
		if errors.Is(err, servicemon.ErrSdnRouteExists) {
			logger.Debug().Println(pkgName, "skip existing SDN route to", destination)
			return nil, nil
		}
		return routestatus.NewEntry(destination, err),
			nil
	}

	routeConflict, conflictIfName := netcfg.RouteConflict(destination)
	if routeConflict {
		// Route already exists. So we have 2 options here
		if strings.HasPrefix(conflictIfName, env.InterfaceNamePrefix) {
			// If route is via other Syntropy interface - change the route to this interface
			// TODO: in future optimise this, and if existing link is legal - try not to change it
			logger.Info().Println(pkgName, "Route update ", destination, " via ", netpath.Gateway, "/", netpath.Ifname)
			err = netcfg.RouteReplace(netpath.Ifname, "", destination)
			if err != nil {
				logger.Error().Println(pkgName, "route update error:", err)
				return routestatus.NewEntry(destination, err), nil
			}
		} else {
			// If route is not via SYNTROPY - inform error
			err = fmt.Errorf("route to %s conflict: wanted via %s exists on %s",
				destination, netpath.Gateway, conflictIfName)
			logger.Error().Println(pkgName, "route add error:", err)
			return routestatus.NewEntry(destination, err), nil
		}

	} else {
		// clean case - no route conflict. Simply add the route
		logger.Info().Println(pkgName, "Route add ", destination, " via ", netpath.Gateway, "/", netpath.Ifname)
		err = netcfg.RouteAdd(netpath.Ifname, "", destination)
		if err != nil {
			logger.Error().Println(pkgName, "route add error:", err)
			return routestatus.NewEntry(destination, err), nil
		}
	}

	return routestatus.NewEntry(destination, err),
		peeradata.NewEntry(0, netpath.ConnectionID, netpath.GroupID)
}

func (r *Router) ServiceDel(netpath *common.SdnNetworkPath, destination string) *routestatus.Entry {
	routesGroup, ok := r.find(netpath.GroupID)
	if !ok {
		// Was asked to delete non-existing service route.
		// So its like I've done what I was asked - do not disturb caller
		logger.Warning().Printf("%s delete service route to %s: route group %d does not exist\n",
			pkgName, destination, netpath.GroupID)
		return routestatus.NewEntry(destination, nil)
	}
	sm := routesGroup.serviceMonitor

	sm.Del(netpath, destination)

	logger.Info().Println(pkgName, "Route delete ", destination, " via ", netpath.Gateway)

	err := netcfg.RouteDel(netpath.Ifname, destination)
	if err != nil {
		logger.Error().Println(pkgName, destination, "route delete error", err)
	}

	return routestatus.NewEntry(destination, err)

}
