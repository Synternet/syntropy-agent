package peerdata

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/slock"
)

const cmd = "IFACES_PEERS_BW_DATA"
const pkgName = "Peer_Data. "

const (
	periodInit = time.Second
	periodRun  = time.Minute
)

type peerDataEntry struct {
	PublicKey  string  `json:"public_key"`
	IP         string  `json:"internal_ip"`
	Handshake  string  `json:"last_handshake,omitempty"`
	KeepAllive int     `json:"keep_alive_interval"`
	Latency    float32 `json:"latency_ms,omitempty"`
	Loss       float32 `json:"packet_loss"`
	Status     string  `json:"status"`
	Reason     string  `json:"status_reason,omitempty"`
}

type ifaceBwEntry struct {
	IfName     string             `json:"iface"`
	PublicKey  string             `json:"iface_public_key"`
	Peers      []*peerDataEntry   `json:"peers"`
	channel    chan *ifaceBwEntry `json:"-"`
	pingClient multiping.PingClient
}

type peerBwData struct {
	common.MessageHeader
	Data []ifaceBwEntry `json:"data"`
}

type wgPeerWatcher struct {
	slock.AtomicServiceLock
	writer  io.Writer
	wg      *swireguard.Wireguard
	ticker  *time.Ticker
	timeout time.Duration
	stop    chan bool
}

func New(writer io.Writer, wgctl *swireguard.Wireguard) common.Service {
	return &wgPeerWatcher{
		wg:      wgctl,
		writer:  writer,
		timeout: periodInit,
		stop:    make(chan bool),
	}
}

func (ie *ifaceBwEntry) PingProcess(pr []multiping.PingResult) {
	// PeerMonitor (as PingClient interface) also needs to process these ping result
	ie.pingClient.PingProcess(pr)

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
	ie.channel <- ie
}

func (obj *wgPeerWatcher) execute() error {
	wg := obj.wg
	resp := peerBwData{}
	resp.ID = env.MessageDefaultID
	resp.MsgType = cmd

	wgdevs, err := wg.Devices()
	if err != nil {
		return err
	}

	count := len(wgdevs)

	// If no interfaces are created yet - I send nothing to controller and wait a short time
	// When interfaces are created - switch to less frequently check
	if count == 0 {
		if obj.timeout != periodInit {
			obj.timeout = periodInit
			obj.ticker.Reset(obj.timeout)
		}
		return nil
	} else if obj.timeout != periodRun {
		obj.timeout = periodRun
		obj.ticker.Reset(obj.timeout)
	}

	// The pinger runs in background, so I will create a channel with number of interfaces
	// And pinger callback will put interface entry (with peers ping info) into channel
	c := make(chan *ifaceBwEntry, count)

	for _, wgdev := range wgdevs {
		ifaceData := ifaceBwEntry{
			IfName:     wgdev.Name,
			PublicKey:  wgdev.PublicKey.String(),
			Peers:      []*peerDataEntry{},
			channel:    c,
			pingClient: wg.PeersMonitor(),
		}
		ping := multiping.New(&ifaceData)

		for _, p := range wgdev.Peers {
			if len(p.AllowedIPs) == 0 {
				continue
			}
			ip := p.AllowedIPs[0]
			ping.AddHost(ip.IP.String())

			var lastHandshake string
			if !p.LastHandshakeTime.IsZero() {
				lastHandshake = p.LastHandshakeTime.Format(env.TimeFormat)
			}

			ifaceData.Peers = append(ifaceData.Peers,
				&peerDataEntry{
					PublicKey:  p.PublicKey.String(),
					IP:         ip.IP.String(),
					Handshake:  lastHandshake,
					KeepAllive: int(p.PersistentKeepaliveInterval.Seconds()),
				})
		}

		ping.Ping()
	}

	for count > 0 {
		entry := <-c
		if len(entry.Peers) > 0 {
			resp.Data = append(resp.Data, *entry)
		}
		count--
	}
	close(c)

	if len(resp.Data) > 0 {
		resp.Now()
		raw, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		obj.writer.Write(raw)
	}

	return nil
}

func (obj *wgPeerWatcher) Name() string {
	return cmd
}

func (obj *wgPeerWatcher) Start() error {
	if !obj.TryLock() {
		return fmt.Errorf("%s is already running", pkgName)
	}

	obj.ticker = time.NewTicker(periodInit)
	go func() {
		for {
			select {
			case <-obj.stop:
				return
			case <-obj.ticker.C:
				obj.execute()

			}
		}
	}()
	return nil
}

func (obj *wgPeerWatcher) Stop() error {
	// Cannot stop not running instance
	if !obj.TryUnlock() {
		return fmt.Errorf("%s is not running", pkgName)

	}

	obj.ticker.Stop()
	obj.stop <- true

	return nil
}
