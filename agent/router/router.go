package router

import (
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
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

func (r *Router) RouteAdd(ifname string, gw string, ips ...string) error {
	for _, ip := range ips {
		newroute := &routeEntry{
			gw:    gw,
			iface: ifname,
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
			continue
		}

		logger.Info().Println(pkgName, "Route add ", ip, " via ", gw)
		err := netcfg.RouteAdd(ifname, gw, ip)
		if err != nil {
			logger.Error().Println(pkgName, "route add error", err)
		}

	}
	return nil
}

func (r *Router) RouteDel(ifname string, ips ...string) error {
	// TODO cleanup routes tree
	return netcfg.RouteDel(ifname, ips...)
}

func (r *Router) Reroute(newgw string) error {
	for dest, routes := range r.routes {
		if routes.Count() <= 1 {
			// cannot do smart routing on only one route list
			continue
		}

		for idx, route := range routes.list {
			if newgw == route.gw {
				if idx == routes.active {
					break
				}
				logger.Info().Println(pkgName, "new route", dest, newgw)
				routes.active = idx
				err := netcfg.RouteReplace(route.iface, newgw, dest)
				if err != nil {
					logger.Error().Println(pkgName, "route replace error", err)
				}
			}
		}
	}
	return nil
}
