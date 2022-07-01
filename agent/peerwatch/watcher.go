package peerwatch

import (
	"context"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"io"
	"net/netip"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/mole"
	"github.com/SyntropyNet/syntropy-agent/agent/netstats"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
)

const (
	cmd     = "PEER_WATCHER"
	pkgName = "PeerWatcher. "
)

type wgPeerWatcher struct {
	writer             io.Writer
	mole               *mole.Mole
	pinger             *multiping.MultiPing
	pingData           *multiping.PingData
	counter            uint
	controlerSendCount uint
}

func New(writer io.Writer, m *mole.Mole, p *multiping.MultiPing) common.Service {
	return &wgPeerWatcher{
		mole:               m,
		writer:             writer,
		pinger:             p,
		pingData:           multiping.NewPingData(),
		controlerSendCount: uint(time.Minute / config.PeerCheckTime()),
	}
}

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

func (obj *wgPeerWatcher) execute(ctx context.Context) error {
	// Update swireguard cached peers statistics
	wgdevs := obj.mole.Wireguard().Devices()
	resp := netstats.NewMessage()
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

			if !p.Stats.LastHandshake.IsZero() {
				// add the peer with handshake to ping list
				pingData.Add(p.AllowedIPs[0].Addr())
			}
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

	// I need these ping results in other places as well
	// SDN rerouting also depends on these pings. Thus I need to ping often
	// But controller does not need this information so oftern. That's why this throtling is here
	obj.counter++
	if obj.counter >= obj.controlerSendCount {
		// reset counter
		obj.counter = 0

		// Peer stats needs to be calculated in same intervals as message send
		obj.mole.Wireguard().PeerStatsUpdate()

		// prepare message to controller
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

				if !p.AllowedIPs[0].IsValid() {
					continue
				}

				// Format message to controller
				entry := &netstats.PeerDataEntry{
					ConnectionID: p.ConnectionID,
					GroupID:      p.GroupID,
					PublicKey:    p.PublicKey,
					IP:           p.AllowedIPs[0].Addr().String(),
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

				ifaceData.Peers = append(ifaceData.Peers, entry)
			}

			resp.Data = append(resp.Data, ifaceData)
		}

		// Fill message with ping statistics
		resp.PingProcess(obj.pingData)

		// Send peers statistics to controller
		err := resp.Send(obj.writer)
		if err != nil {
			logger.Debug().Println(pkgName, "Send error", err)
		}

		// Reset statistics for the next ping period
		obj.pingData.Reset()
	}

	return nil
}

func (obj *wgPeerWatcher) Name() string {
	return cmd
}

func (obj *wgPeerWatcher) Run(ctx context.Context) error {
	go func() {
		ticker := time.NewTicker(config.PeerCheckTime())
		// initial peer stats
		obj.mole.Wireguard().PeerStatsUpdate()
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Debug().Println(pkgName, "stopping", cmd)
				return
			case <-ticker.C:
				obj.execute(ctx)
			}
		}
	}()
	return nil
}
