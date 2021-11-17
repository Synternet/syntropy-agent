package servicemon

import (
	"strings"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/netcfg"
)

func (sm *ServiceMonitor) Reroute(newgw string) []peeradata.PeerActiveDataEntry {
	errIPs := []string{}
	ret := []peeradata.PeerActiveDataEntry{}

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
						peeradata.PeerActiveDataEntry{
							PreviousConnID: oldRoute.connectionID,
							ConnectionID:   newRoute.connectionID,
							GroupID:        newRoute.groupID,
							Timestamp:      time.Now().Format(env.TimeFormat),
						})
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
