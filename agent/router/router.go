package router

import (
	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
)

/**
 * Hugh.... I'm a little in doubt here...
 * I do not like GOs standard `net` package IP structs and interfaces
 * vishvananda's netlink package is either too low level or tries reusing net packages interfaces
 * An option could be to use tailscale's inet.af/netaddr, but this needs more investigation
 * what benefits tradeoffs we will get
 * So for now lets stick to plain strings
 *
 * Some good news. New 1.18 GO will include net/netip in stdlib,
 * which is almost same tailscale's inet.af/netaddr
 * https://sebastian-holstein.de/post/2021-11-08-go-1.18-features/
 * So lets wait till February 2022 and finaly fix this.
 * TODO ^^^^
 **/

func (r *Router) RouteAdd(netpath *common.SdnNetworkPath, dest []string) ([]common.RouteResult, []peeradata.PeerActiveDataEntry) {
	const defaultRouteIP = "0.0.0.0/0"
	res := []common.RouteResult{}
	pad := []peeradata.PeerActiveDataEntry{}

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
			resEntry, padEntry := r.ServiceAdd(netpath, ip)
			if resEntry != nil {
				res = append(res, *resEntry)
			}
			if padEntry != nil {
				pad = append(pad, *padEntry)
			}
		}
	}

	return res, pad
}

func (r *Router) RouteDel(netpath *common.SdnNetworkPath, ips []string) []common.RouteResult {
	res := []common.RouteResult{}
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
			rres := r.ServiceDel(netpath, ip)
			if rres != nil {
				res = append(res, *rres)
			}
		}
	}

	return res
}
