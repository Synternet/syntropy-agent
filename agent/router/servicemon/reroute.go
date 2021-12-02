package servicemon

import (
	"strings"

	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

func (sm *ServiceMonitor) Reroute(newgw string) []*peeradata.Entry {
	errIPs := []string{}
	ret := []*peeradata.Entry{}

	sm.Lock()
	defer sm.Unlock()

	for dest, routes := range sm.routes {
		if routes.Count() <= 1 {
			// cannot do smart routing on only one route list
			continue
		}

		for idx, newRoute := range routes.list {
			if newgw == newRoute.gateway {
				if idx == routes.active {
					break
				}
				oldRoute := routes.list[routes.active]
				logger.Info().Printf("%s SDN route change to %s via %s [%s:%d]\n",
					pkgName, dest, newgw, newRoute.ifname, newRoute.groupID)
				routes.active = idx
				err := netcfg.RouteReplace(newRoute.ifname, newgw, dest)
				if err == nil {
					ret = append(ret,
						peeradata.NewEntry(oldRoute.connectionID, newRoute.connectionID, newRoute.groupID))
				} else {
					logger.Error().Println(pkgName, err)
					errIPs = append(errIPs, dest)
				}
			}
		}
	}

	if len(errIPs) > 0 {
		logger.Error().Printf("%s could not change routes to %s via %s\n",
			pkgName, strings.Join(errIPs, ","), newgw)
	}

	return ret
}
