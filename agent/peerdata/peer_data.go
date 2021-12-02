package peerdata

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
)

type wgPeerWatcher struct {
	writer      io.Writer
	wg          *swireguard.Wireguard
	timeout     time.Duration
	pinger      *multiping.MultiPing
	pingData    *multiping.PingData
	pingClients []multiping.PingClient
	counter     int
}

func New(writer io.Writer, wgctl *swireguard.Wireguard,
	p *multiping.MultiPing, pcl ...multiping.PingClient) common.Service {
	return &wgPeerWatcher{
		wg:          wgctl,
		writer:      writer,
		timeout:     periodInit,
		pinger:      p,
		pingData:    multiping.NewPingData(),
		pingClients: pcl,
	}
}

func (obj *wgPeerWatcher) PingProcess(pr *multiping.PingData) {
	// PingClients (actually PeerMonitor instance) also needs to process these ping result
	for _, pc := range obj.pingClients {
		pc.PingProcess(pr)
	}
}

func (obj *wgPeerWatcher) execute(ctx context.Context, ticker *time.Ticker) error {
	wg := obj.wg

	// Update swireguard cached peers statistics
	wg.PeerStatsUpdate()

	resp := newMsg()
	resp.ID = env.MessageDefaultID
	resp.MsgType = cmd

	wgdevs := wg.Devices()

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

	// Clean residual data from last ping
	obj.pingData.Flush()

	for _, wgdev := range wgdevs {
		ifaceData := ifaceBwEntry{
			IfName:    wgdev.IfName,
			PublicKey: wgdev.PublicKey,
			Peers:     []*peerDataEntry{},
		}

		for _, p := range wgdev.Peers() {
			if len(p.AllowedIPs) == 0 {
				continue
			}

			// AllowedIPs has cidr notation. I need only the address for pinging.
			// TODO: research usable IP address struct
			ip := strings.Split(p.AllowedIPs[0], "/")[0]
			if len(ip) == 0 {
				continue
			}
			obj.pingData.Add(ip)

			var lastHandshake string
			if !p.Stats.LastHandshake.IsZero() {
				lastHandshake = p.Stats.LastHandshake.Format(env.TimeFormat)
			}

			ifaceData.Peers = append(ifaceData.Peers,
				&peerDataEntry{
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
	if obj.pingData.Count() == 0 {
		return nil
	}

	// Ping the connected peers
	obj.pinger.Ping(obj.pingData)
	// Some other users (e.g. PeerMonitor) are also interested in these results
	// NOTE: optimisation - ping statistics are not yet added to IFACES_PEERS_BW_DATA message (resp)
	obj.PingProcess(obj.pingData)

	// I need these ping results in other places as well
	// SDN rerouting also depends on these pings. Thus I need to ping often
	// But controller does not need this information so oftern. That's why this throtling is here
	obj.counter++
	if obj.counter >= controllerSendPeriod {
		obj.counter = 0

		resp.Now()
		// Fill message with ping statistics
		resp.PingProcess(obj.pingData)

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
