package router

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/netcfg"
)

/**
 * Hugh.... I'm a little in doubt here...
 * I do not like GOs standard `net` package IP structs and interfaces
 * vishvananda's netlink package is either too low level or tries reusing net packages interfaces
 * An option could be to use tailscale's inet.af/netaddr, but this needs more investigation
 * what benefits tradeoffs we will get
 * So for now lets stick to plain strings (TODO)
 **/

func (r *Router) RouteAdd(netpath *common.SdnNetworkPath, dest ...string) error {
	errIPs := []string{}

	for _, ip := range dest {
		newroute := &routeEntry{
			ifname:  netpath.Ifname,
			gateway: netpath.Gateway,
			id:      netpath.ID,
		}
		if r.routes[ip] == nil {
			r.routes[ip] = &routeList{
				// when adding new destination - always start with the first route active
				// Yes, I know this is zero by default, but I wanted it to be explicitely clear
				active: 0,
			}
		}
		r.routes[ip].Add(newroute)

		if r.routes[ip].Count() > 1 {
			// TODO: I think I should inform controller about route errors
			logger.Debug().Println(pkgName, "skip existing SDN route to", ip)
			continue
		}

		if netcfg.RouteExists(ip) {
			logger.Warning().Println(pkgName, "skip existing route to ", ip)
			errIPs = append(errIPs, ip)
			continue
		}

		logger.Info().Println(pkgName, "Route add ", ip, " via ", netpath.Gateway)
		err := netcfg.RouteAdd(netpath.Ifname, netpath.Gateway, ip)
		if err != nil {
			logger.Error().Println(pkgName, "route add error", err)
			errIPs = append(errIPs, ip)
		}

	}

	if len(errIPs) > 0 {
		return fmt.Errorf("could not add routes to %s", strings.Join(errIPs, ","))
	}

	return nil
}

func (r *Router) RouteDel(netpath *common.SdnNetworkPath, ips ...string) error {
	errIPs := []string{}
	for _, ip := range ips {
		if r.routes[ip] != nil {
			delete(r.routes, ip)
			err := netcfg.RouteDel(netpath.Ifname, ip)
			if err != nil {
				errIPs = append(errIPs, ip)
				logger.Error().Println(pkgName, ip, "route delete error", err)
			}

		}
	}

	if len(errIPs) > 0 {
		return fmt.Errorf("could not delete routes to %s", strings.Join(errIPs, ","))
	}

	return nil
}

func (r *Router) Reroute(newgw string) error {
	errIPs := []string{}
	resp := newRespMsg()

	for dest, routes := range r.routes {
		if routes.Count() <= 1 {
			// cannot do smart routing on only one route list
			continue
		}

		for idx, route := range routes.list {
			if newgw == route.gateway {
				if idx == routes.active {
					break
				}
				logger.Info().Printf("%s change route to %s via %s [id:%d]\n", pkgName, dest, newgw, route.id)
				logger.Info().Println(pkgName, idx, routes.active)
				log.Println(routes)
				routes.active = idx
				err := netcfg.RouteReplace(route.ifname, newgw, dest)
				if err == nil {
					resp.Data = append(resp.Data,
						peerActiveDataEntry{
							ConnectionID: route.id,
							Timestamp:    time.Now().Format(env.TimeFormat),
						})
				} else {
					logger.Error().Println(pkgName, err)
					errIPs = append(errIPs, dest)
				}
			}
		}
	}

	// TODO thing about sending errors to controller
	if len(resp.Data) > 0 {
		resp.Now()
		raw, err := json.Marshal(resp)
		if err != nil {
			return err
		}

		r.writer.Write(raw)
	}

	if len(errIPs) > 0 {
		return fmt.Errorf("could not change routes to %s via %s", strings.Join(errIPs, ","), newgw)
	}

	return nil
}
