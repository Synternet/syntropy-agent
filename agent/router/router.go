package router

import (
	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/generic/router"
)

/**
 * Hugh.... I'm a little in doubt here...
 * I do not like GOs standard `net` package IP structs and interfaces
 * vishvananda's netlink package is either too low level or tries reusing net packages interfaces
 * An option could be to use tailscale's inet.af/netaddr, but this needs more investigation
 * what benefits tradeoffs we will get
 * So for now lets stick to plain strings (TODO)
 **/

func (r *Router) RouteAdd(netpath *router.SdnNetworkPath, dest []string) []router.RouteResult {
	const defaultRouteIP = "0.0.0.0/0"
	res := []router.RouteResult{}

	r.Lock()
	defer r.Unlock()

	for idx, ip := range dest {
		// A very dumb protection from "bricking" servers by adding default routes
		// Allow add default routes only for configured VPN_CLIENT
		// TODO: there are dosens other ways to act as default route, without 0.0.0.0 IP
		if !config.IsVPNClient() && ip == defaultRouteIP {
			logger.Warning().Println(pkgName, "ignored default route for non configured VPN client")
			continue
		}

		// Some hidden business logic here:
		// Controller sends Allowed_IPs as follows:
		// first entry (index=0) is its WG tunnel peers internal ip ==> need to add host route
		// all other entries are peers LANs (docker, etc) services IPs, that should have SDN routing on them
		if idx == 0 {
			r.PeerAdd(netpath, ip)
		} else {
			res = append(res, r.ServiceAdd(netpath, ip))
		}
	}

	return res
}

func (r *Router) RouteDel(netpath *router.SdnNetworkPath, ips []string) []router.RouteResult {
	res := []router.RouteResult{}
	r.Lock()
	defer r.Unlock()

	for idx, ip := range ips {
		// Some hidden business logic here:
		// Controller sends Allowed_IPs as follows:
		// first entry (index=0) is its WG tunnel peers internal ip ==> need to add host route
		// all other entries are peers LANs (docker, etc) services IPs, that should have SDN routing on them
		if idx == 0 {
			r.PeerDel(netpath, ip)
		} else {
			res = append(res, r.ServiceDel(netpath, ip))
		}
	}

	return res
}
