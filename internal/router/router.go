package router

import (
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/netcfg"
)

const pkgName = "Router. "

/**
 * Hugh.... I'm a little in doubt here...
 * I do not like GOs standard `net` package IP structs and interfaces
 * vishvananda's netlink package is either too low level or tries reusing net packages interfaces
 * An option could be to use tailscale's inet.af/netaddr, but this needs more investigation
 * what benefits tradeoffs we will get
 * So for now lets stick to plain strings (TODO)
 **/

type route struct {
	gw    string
	iface string
	flags bits
}

type Router struct {
	routes map[string][]*route
}

func New() *Router {
	r := Router{}
	r.routes = make(map[string][]*route)

	return &r
}

func (r *Router) RouteAdd(ifname string, gw string, ips ...string) error {
	for _, ip := range ips {
		newroute := &route{
			gw:    gw,
			iface: ifname,
			flags: agent,
		}
		r.routes[ip] = append(r.routes[ip], newroute)

		if len(r.routes[ip]) > 1 {
			logger.Debug().Println(pkgName, "skip existing SDN route to", ip)
			continue
		}

		if netcfg.RouteExists(ip) {
			logger.Warning().Println(pkgName, "skip existing route to ", ip)
			continue
		}

		newroute.flags.set(active)
		err := netcfg.RouteAdd(ifname, gw, ip)
		if err != nil {
			newroute.flags.clear(active)
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
		if len(routes) <= 1 {
			// cannot do smart routing on only one route list
			continue
		}

		for _, route := range routes {
			if newgw == route.gw {
				if route.flags.has(active) {
					break
				}
				logger.Info().Println(pkgName, "new route", dest, newgw)
				route.flags.set(active)
				err := netcfg.RouteReplace(route.iface, newgw, dest)
				if err != nil {
					route.flags.clear(active)
					logger.Error().Println(pkgName, "route replace error", err)
				}
			} else {
				route.flags.clear(active)
			}
		}
	}
	return nil
}
