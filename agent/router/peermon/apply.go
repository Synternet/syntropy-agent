package peermon

import (
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/peerlist"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

func (pm *PeerMonitor) Apply() error {
	deleteIPs := []netip.Prefix{}

	pm.peerList.Iterate(func(ip netip.Prefix, peer *peerlist.PeerInfo) {
		if peer.HasFlag(peerlist.PifDisabled) {
			if peer.HasFlag(peerlist.PifDelPending) {
				deleteIPs = append(deleteIPs, ip)
			} else {
				logger.Warning().Println(pkgName, "Apply ignored conflicting IP", ip)
			}
			return
		} else if peer.HasFlag(peerlist.PifAddPending) {
			logger.Debug().Println(pkgName, "Add peer route to", ip)
			err := netcfg.RouteAdd(peer.Ifname, nil, &ip)
			if err != nil {
				logger.Error().Println(pkgName, ip, "route add error:", err)
			}
			peer.ResetFlags()

		} else if peer.HasFlag(peerlist.PifDelPending) {
			logger.Debug().Println(pkgName, "Delete peer route to", ip)
			err := netcfg.RouteDel(peer.Ifname, &ip)
			if err != nil {
				logger.Error().Println(pkgName, ip, "route delete error", err)
			}
			peer.ResetFlags()
			deleteIPs = append(deleteIPs, ip)
		}

	})

	pm.peerList.Delete(deleteIPs...)

	return nil
}

func (pm *PeerMonitor) ResolveIpConflict(isIPconflict func(netip.Prefix, int) bool) (count int) {
	pm.peerList.Iterate(func(ip netip.Prefix, peer *peerlist.PeerInfo) {
		if peer.HasFlag(peerlist.PifDisabled) {
			// check if IP conflict still present
			if !isIPconflict(ip, pm.groupID) {
				// clear disabled flag and increment updated peers count
				peer.ClearFlag(peerlist.PifDisabled)
				count++
			}
		}
	})

	return count
}

func (pm *PeerMonitor) Flush() {
	pm.peerList.Iterate(func(ip netip.Prefix, peer *peerlist.PeerInfo) {
		peer.SetFlag(peerlist.PifDelPending)
	})
}

func (pm *PeerMonitor) Close() error {
	// Cleanup peers on exit
	// Reuse Flush and Apply functions
	pm.Flush()
	return pm.Apply()
}
