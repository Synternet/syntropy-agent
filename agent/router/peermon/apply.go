package peermon

import (
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

func (pm *PeerMonitor) Apply() error {
	deleteIPs := []netip.Prefix{}

	for ip, peer := range pm.peerList {
		if peer.HasFlag(pifDisabled) {
			if peer.HasFlag(pifDelPending) {
				deleteIPs = append(deleteIPs, ip)
			} else {
				logger.Warning().Println(pkgName, "Apply ignored conflicting IP", ip)
			}
			continue
		} else if peer.HasFlag(pifAddPending) {
			logger.Debug().Println(pkgName, "Add peer route to", ip)
			err := netcfg.RouteAdd(peer.ifname, nil, &ip)
			if err != nil {
				logger.Error().Println(pkgName, ip, "route add error:", err)
			}
			peer.flags = pifNone

		} else if peer.HasFlag(pifDelPending) {
			logger.Debug().Println(pkgName, "Delete peer route to", ip)
			err := netcfg.RouteDel(peer.ifname, &ip)
			if err != nil {
				logger.Error().Println(pkgName, ip, "route delete error", err)
			}
			peer.flags = pifNone
			deleteIPs = append(deleteIPs, ip)
		}

	}

	for _, ip := range deleteIPs {
		delete(pm.peerList, ip)
	}

	return nil
}

func (pm *PeerMonitor) ResolveIpConflict(isIPconflict func(netip.Prefix, int) bool) (count int) {
	for ip, peer := range pm.peerList {
		if peer.HasFlag(pifDisabled) {
			// check if IP conflict still present
			if !isIPconflict(ip, pm.groupID) {
				// clear disabled flag and increment updated peers count
				peer.flags &= ^pifDisabled
				count++
			}
		}
	}
	return count
}

func (pm *PeerMonitor) Flush() {
	for _, peer := range pm.peerList {
		peer.flags |= pifDelPending
	}
}

func (pm *PeerMonitor) Close() error {
	// Cleanup peers on exit
	// Reuse Flush and Apply functions
	pm.Flush()
	return pm.Apply()
}
