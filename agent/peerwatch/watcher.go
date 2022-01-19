package peerwatch

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/mole"
	"github.com/SyntropyNet/syntropy-agent/agent/netstats"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
)

const (
	cmd     = "PEER_WATCHER"
	pkgName = "PeerWatcher. "
)

const (
	periodInit           = time.Second
	periodRun            = time.Second * 5 // ping every 5 seconds
	controllerSendPeriod = 12              // reduce messages to controller to every minute
)

type wgPeerWatcher struct {
	writer   io.Writer
	mole     *mole.Mole
	timeout  time.Duration
	pinger   *multiping.MultiPing
	pingData *multiping.PingData
	counter  int
}

func New(writer io.Writer, m *mole.Mole, p *multiping.MultiPing) common.Service {
	return &wgPeerWatcher{
		mole:     m,
		writer:   writer,
		timeout:  periodInit,
		pinger:   p,
		pingData: multiping.NewPingData(),
	}
}

func (obj *wgPeerWatcher) PingProcess(pr *multiping.PingData) {
	// PeerMonitor instance (member of Router) also needs to process these ping result
	obj.mole.Router().PingProcess(pr)

	// Now merge ping results
	obj.pingData.Append(pr)

	// finally cleanup removed peers
	removeIPs := []string{}
	obj.pingData.Iterate(func(ip string, val multiping.PingStats) {
		_, found := pr.Get(ip)
		if !found {
			// peer not found - add to remove list
			removeIPs = append(removeIPs, ip)
		}
	})
	// Remove deleted peers
	obj.pingData.Del(removeIPs...)
}

func (obj *wgPeerWatcher) execute(ctx context.Context, ticker *time.Ticker) error {
	// Update swireguard cached peers statistics
	obj.mole.Wireguard().PeerStatsUpdate()
	wgdevs := obj.mole.Wireguard().Devices()
	resp := netstats.NewMessage()
	pingData := multiping.NewPingData()

	// If no interfaces are created yet - I send nothing to controller and wait a short time
	// When interfaces are created - switch to less frequently check
	if len(wgdevs) == 0 {
		if obj.timeout != periodInit {
			obj.timeout = periodInit
			ticker.Reset(obj.timeout)
		}
		return nil
	} else if obj.timeout != periodRun {
		obj.timeout = periodRun
		ticker.Reset(obj.timeout)
	}

	// prepare peers ping list and message to controller
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

			// AllowedIPs has cidr notation. I need only the address for pinging.
			ip := strings.Split(p.AllowedIPs[0], "/")[0]
			if len(ip) == 0 {
				continue
			}

			// add peers to ping list
			pingData.Add(ip)

			var lastHandshake string
			if !p.Stats.LastHandshake.IsZero() {
				lastHandshake = p.Stats.LastHandshake.Format(env.TimeFormat)
			}

			ifaceData.Peers = append(ifaceData.Peers,
				&netstats.PeerDataEntry{
					ConnectionID: p.ConnectionID,
					GroupID:      p.GroupID,
					PublicKey:    p.PublicKey,
					IP:           ip,
					Handshake:    lastHandshake,
					KeepAllive:   int(swireguard.KeepAlliveDuration.Seconds()),
					RxBytes:      p.Stats.RxBytes,
					TxBytes:      p.Stats.TxBytes,
					RxSpeed:      p.Stats.RxSpeedMbps,
					TxSpeed:      p.Stats.TxSpeedMbps,
				})
		}
		resp.Data = append(resp.Data, ifaceData)
	}

	// pingData now contains all connected peers on all interfaces
	// Perform ping and process results, if I have any connected peers
	// Do nothing if no peers are configured
	if pingData.Count() == 0 {
		return nil
	}

	// Ping the connected peers
	obj.pinger.Ping(pingData)
	// Some other users (e.g. PeerMonitor) are also interested in these results
	// NOTE: optimisation - ping statistics are not yet added to IFACES_PEERS_BW_DATA message (resp)
	obj.PingProcess(pingData)

	// I need these ping results in other places as well
	// SDN rerouting also depends on these pings. Thus I need to ping often
	// But controller does not need this information so oftern. That's why this throtling is here
	obj.counter++
	if obj.counter >= controllerSendPeriod {
		obj.counter = 0

		// Fill message with ping statistics
		resp.PingProcess(obj.pingData)

		// Reset statistics for the next ping period
		obj.pingData.Reset()

		// Send peers statistics to controller
		resp.Now()
		raw, err := json.Marshal(resp)
		if err != nil {
			logger.Error().Println(pkgName, "json", err)
			return err
		}

		logger.Debug().Println(pkgName, "Sending: ", string(raw))
		obj.writer.Write(raw)
	}

	return nil
}

func (obj *wgPeerWatcher) Name() string {
	return cmd
}

func (obj *wgPeerWatcher) Run(ctx context.Context) error {
	go func() {
		ticker := time.NewTicker(periodInit)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				obj.execute(ctx, ticker)
			}
		}
	}()
	return nil
}
