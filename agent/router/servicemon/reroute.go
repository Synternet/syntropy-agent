package servicemon

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/agent/router/ipadmsg"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/netcfg"
)

func (sm *ServiceMonitor) Reroute(newgw string) error {
	errIPs := []string{}
	resp := ipadmsg.NewMessage()

	sm.Lock()

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
				routes.Print()
				routes.active = idx
				err := netcfg.RouteReplace(newRoute.ifname, newgw, dest)
				if err == nil {
					resp.Data = append(resp.Data,
						ipadmsg.PeerActiveDataEntry{
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

	sm.Unlock()

	if len(resp.Data) > 0 {
		resp.Now()
		raw, err := json.Marshal(resp)
		if err != nil {
			return err
		}

		logger.Debug().Println(pkgName, "Sending: ", string(raw))
		sm.writer.Write(raw)
	}

	if len(errIPs) > 0 {
		return fmt.Errorf("could not change routes to %s via %s", strings.Join(errIPs, ","), newgw)
	}

	return nil
}
