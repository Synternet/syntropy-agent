package peerdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/scontext"
)

const cmd = "IFACES_PEERS_BW_DATA"
const pkgName = "Peer_Data. "

const (
	periodInit           = time.Second
	periodRun            = time.Second * 5 // ping every 5 seconds
	controllerSendPeriod = 12              // reduce messages to controller to every minute
)

type peerDataEntry struct {
	PublicKey    string  `json:"public_key"`
	IP           string  `json:"internal_ip"`
	Handshake    string  `json:"last_handshake,omitempty"`
	KeepAllive   int     `json:"keep_alive_interval"`
	Latency      float32 `json:"latency_ms,omitempty"`
	Loss         float32 `json:"packet_loss"`
	Status       string  `json:"status"`
	Reason       string  `json:"status_reason,omitempty"`
	RxBytes      int64   `json:"rx_bytes"`
	TxBytes      int64   `json:"tx_bytes"`
	RxSpeed      float32 `json:"rx_speed_mbps"`
	TxSpeed      float32 `json:"tx_speed_mbps"`
	ConnectionID int     `json:"connection_id"`
	GroupID      int     `json:"connection_group_id"`
}

type ifaceBwEntry struct {
	IfName      string           `json:"iface"`
	PublicKey   string           `json:"iface_public_key"`
	Peers       []*peerDataEntry `json:"peers"`
	wait        *sync.WaitGroup  `json:"-"`
	pingClients []multiping.PingClient
}

type peerBwData struct {
	common.MessageHeader
	Data []ifaceBwEntry `json:"data"`
}

type wgPeerWatcher struct {
	writer      io.Writer
	wg          *swireguard.Wireguard
	timeout     time.Duration
	ctx         scontext.StartStopContext
	pingClients []multiping.PingClient
	counter     int
}

func New(ctx context.Context, writer io.Writer, wgctl *swireguard.Wireguard, pcl ...multiping.PingClient) common.Service {
	return &wgPeerWatcher{
		wg:          wgctl,
		writer:      writer,
		timeout:     periodInit,
		ctx:         scontext.New(ctx),
		pingClients: pcl,
	}
}

func (ie *ifaceBwEntry) PingProcess(pr []multiping.PingResult) {
	defer ie.wait.Done()

	// PingClients (actually PeerMonitor instance) also needs to process these ping result
	for _, pc := range ie.pingClients {
		pc.PingProcess(pr)
	}

	var entry *peerDataEntry

	// format results for controler
	for _, pingres := range pr {
		entry = nil
		for _, e := range ie.Peers {
			if e.IP == pingres.IP {
				entry = e
				break
			}
		}

		if entry == nil {
			continue
		}

		entry.Latency = pingres.Latency
		entry.Loss = pingres.Loss

		switch {
		case entry.Loss >= 1:
			entry.Status = "OFFLINE"
			entry.Reason = "Packet loss 100%"
		case entry.Loss >= 0.01 && entry.Loss < 1:
			entry.Status = "WARNING"
			entry.Reason = "Packet loss higher than 1%"
		case entry.Latency > 500:
			entry.Status = "WARNING"
			entry.Reason = "Latency higher than 500ms"
		default:
			entry.Status = "CONNECTED"
		}
	}
}

func (obj *wgPeerWatcher) execute(ctx context.Context, ticker *time.Ticker) error {
	wg := obj.wg

	// Update swireguard cached peers statistics
	wg.PeerStatsUpdate()

	resp := peerBwData{}
	resp.ID = env.MessageDefaultID
	resp.MsgType = cmd

	wgdevs := wg.Devices()

	count := len(wgdevs)

	// If no interfaces are created yet - I send nothing to controller and wait a short time
	// When interfaces are created - switch to less frequently check
	if count == 0 {
		if obj.timeout != periodInit {
			obj.timeout = periodInit
			ticker.Reset(obj.timeout)
		}
		return nil
	} else if obj.timeout != periodRun {
		obj.timeout = periodRun
		ticker.Reset(obj.timeout)
	}

	// The pinger runs in background - use a WaitGroup to synchronise
	wait := sync.WaitGroup{}

	for _, wgdev := range wgdevs {
		ifaceData := ifaceBwEntry{
			IfName:      wgdev.IfName,
			PublicKey:   wgdev.PublicKey,
			Peers:       []*peerDataEntry{},
			wait:        &wait,
			pingClients: obj.pingClients,
		}
		ping := multiping.New(ctx, &ifaceData)

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
			ping.AddHost(ip)

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
		if len(ifaceData.Peers) > 0 {
			wait.Add(1)
			resp.Data = append(resp.Data, ifaceData)
			ping.Ping()
		}
	}

	wait.Wait()

	obj.counter++
	// TODO: optimise and do not parse results if not sending
	if obj.counter >= controllerSendPeriod {
		obj.counter = 0
		if len(resp.Data) > 0 {
			resp.Now()
			raw, err := json.Marshal(resp)
			if err != nil {
				logger.Error().Println(pkgName, "json", err)
				return err
			}

			logger.Debug().Println(pkgName, "Sending: ", string(raw))
			obj.writer.Write(raw)
		}
	}

	return nil
}

func (obj *wgPeerWatcher) Name() string {
	return cmd
}

func (obj *wgPeerWatcher) Start() error {
	ctx, err := obj.ctx.CreateContext()
	if err != nil {
		return fmt.Errorf("%s is already running", pkgName)
	}

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

func (obj *wgPeerWatcher) Stop() error {
	// Cannot stop not running instance
	if err := obj.ctx.CancelContext(); err != nil {
		return fmt.Errorf("%s is not running", pkgName)

	}

	return nil
}
