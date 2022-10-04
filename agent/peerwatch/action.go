package peerwatch

import (
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/netstats"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
)

func (obj *wgPeerWatcher) PingProcess(pr *multiping.PingData) {
	// PeerMonitor instance (member of Router) also needs to process these ping result
	obj.mole.Router().PingProcess(pr)

	// Now merge ping results to keep average values for the whole period
	obj.pingData.Append(pr)

	// finally cleanup removed peers
	var removeIPs []netip.Addr
	obj.pingData.Iterate(func(ip netip.Addr, val multiping.PingStats) {
		_, found := pr.Get(ip)
		if !found {
			// peer not found - add to remove list
			removeIPs = append(removeIPs, ip)
		}
	})
	// Remove deleted peers
	obj.pingData.Del(removeIPs...)
}

func (obj *wgPeerWatcher) monitorPeers(wgdevs []*swireguard.InterfaceInfo) error {
	pingData := multiping.NewPingData()

	// prepare peers ping list
	for _, wgdev := range wgdevs {
		for _, p := range wgdev.Peers() {
			if len(p.AllowedIPs) == 0 {
				continue
			}

			if !p.AllowedIPs[0].IsValid() {
				continue
			}

			// add the peer to ping list
			pingData.Add(p.AllowedIPs[0].Addr())
		}
	}
	// pingData now contains all connected peers on all interfaces
	// Perform ping and process results, if I have any connected peers
	// Do nothing if no peers are configured
	if pingData.Count() > 0 {
		// Ping the connected peers
		obj.pinger.Ping(pingData)
	}

	// Some other users (e.g. PeerMonitor) are also interested in these results
	// NOTE: optimisation - ping statistics are not yet added to IFACES_PEERS_BW_DATA message (resp)
	obj.PingProcess(pingData)

	return nil
}

func (obj *wgPeerWatcher) message2controller(wgdevs []*swireguard.InterfaceInfo) error {
	// I need these ping results in other places as well
	// SDN rerouting also depends on these pings. Thus I need to ping often
	// But controller does not need this information so oftern. That's why this throtling is here
	obj.counter++
	if obj.counter < obj.controlerSendCount {
		return nil
	}

	// Update wireguard cached peers statistics
	// Peer stats needs to be calculated in same intervals as message send
	obj.mole.Wireguard().PeerStatsUpdate()

	// prepare message to controller
	resp := netstats.NewMessage()
	for _, wgdev := range wgdevs {
		ifaceData := netstats.IfaceBwEntry{
			IfName:    wgdev.IfName,
			PublicKey: wgdev.PublicKey,
			Peers:     []*netstats.PeerDataEntry{},
		}

		for _, p := range wgdev.Peers() {
			if len(p.AllowedIPs) == 0 {
				continue
			}

			ipAddr := p.AllowedIPs[0]
			if !ipAddr.IsValid() {
				continue
			}

			// Format message to controller
			entry := &netstats.PeerDataEntry{
				ConnectionID: p.ConnectionID,
				GroupID:      p.GroupID,
				PublicKey:    p.PublicKey,
				IP:           ipAddr.Addr().String(),
				KeepAllive:   int(swireguard.KeepAlliveDuration.Seconds()),
				RxBytes:      p.Stats.RxBytesDiff, // Controler is expecting bytes received during report period
				TxBytes:      p.Stats.TxBytesDiff, // Controler is expecting bytes sent during report period
				RxSpeed:      p.Stats.RxSpeedMBps,
				TxSpeed:      p.Stats.TxSpeedMBps,
			}

			if p.Stats.LastHandshake.IsZero() {
				entry.Loss = netstats.PingLoss
			} else {
				entry.Handshake = p.Stats.LastHandshake.Format(env.TimeFormat)
			}

			stats, ok := obj.pingData.Get(netip.Addr(ipAddr.Addr()))
			// All configured peers are pinged in monitorPeers() function
			// But when searching for a peer in ping result may be missing on one case:
			// Config_Info is still in progress (it can take few minutes with big config on low end machine)
			// Then a situation may arrise when ping monitoring already started,
			// and config_info setup was not yet completed
			// In this case during ping data processing wireguard may have more peers configured,
			// and some of these peers are not pinged yet (they we not configured when ping was done)
			// This is no dangerous at all. Ignore these results now and this peer ping will be done next time.
			if ok {
				if stats.Valid() {
					entry.Loss = stats.Loss()
					entry.Latency = stats.Latency()
				} else {
					entry.Loss = netstats.PingLoss
					logger.Warning().Println(pkgName, "Invalid ping stats for", ipAddr)
				}
				// Once again read comment above
				// Add only fully configured and already pinged peers
				ifaceData.Peers = append(ifaceData.Peers, entry)
			}
		}

		resp.Data = append(resp.Data, ifaceData)
	}

	// Reset statistics and counter for the next ping period
	obj.pingData.Reset()
	obj.counter = 0

	// Send peers statistics to controller
	return resp.Send(obj.writer)
}
