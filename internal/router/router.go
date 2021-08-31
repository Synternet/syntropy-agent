package router

import (
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/netcfg"
)

const pkgName = "Router. "

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
			err := netcfg.RouteAdd(ifname, gw, ip)
			if err != nil {
				logger.Error().Println(pkgName, "route add error", err)
			}
		} else {
			logger.Debug().Println(pkgName, "skip existing SDN route to", ip)
		}

	}
	return netcfg.RouteAdd(ifname, gw, ips...)
}

func (r *Router) RouteDel(ifname string, ips ...string) error {
	return netcfg.RouteDel(ifname, ips...)
}
