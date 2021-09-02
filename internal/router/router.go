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
		r.routes[ip] = append(r.routes[ip], &route{
			gw:    gw,
			iface: ifname,
			flags: agent,
		})

		if len(r.routes[ip]) == 1 {
			logger.Info().Println(pkgName, "Route add ", ip, " via ", gw)
			if netcfg.RouteExists(ip) {
				logger.Warning().Println(pkgName, "skip existing route to ", ip)
				continue
			}

			err := netcfg.RouteAdd(ifname, gw, ip)
			if err != nil {
				logger.Error().Println(pkgName, "route add error", err)
			}
		} else {
			logger.Debug().Println(pkgName, "skip existing SDN route to", ip)
		}

	}
	return nil
}

func (r *Router) RouteDel(ifname string, ips ...string) error {
	// TODO cleanup routes tree
	return netcfg.RouteDel(ifname, ips...)
}
