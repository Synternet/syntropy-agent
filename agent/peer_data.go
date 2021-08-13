package agent

import (
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/pinger"
	"github.com/SyntropyNet/syntropy-agent-go/wireguard"
)

const (
	periodInit = time.Second
	periodRun  = time.Minute
)

type peerDataEntry struct {
	IP      string  `json:"internal_ip"`
	Latency int     `json:"latency_ms"`
	Loss    float32 `json:"packet_loss"`
	Status  string  `json:"status"`
	Reason  string  `json:"status_reason,omitempty"`
}

type ifaceBwEntry struct {
	IfName    string             `json:"iface"`
	PublicKey string             `json:"iface_public_key"`
	Peers     []peerDataEntry    `json:"peers"`
	channel   chan *ifaceBwEntry `json:"-"`
}

type peerBwData struct {
	messageHeader
	Data []ifaceBwEntry `json:"data"`
}

// TODO: think of INTERFACE type to fit all background runners.
// Make this object interface. If possible - remove Agent dependency
type WgPeerWatcher struct {
	wg      *wireguard.Wireguard
	writer  io.Writer
	ticker  *time.Ticker
	timeout time.Duration
	stop    chan bool
}

func NewWgPeerWatcher(wgctl *wireguard.Wireguard, writer io.Writer) *WgPeerWatcher {
	return &WgPeerWatcher{
		wg:      wgctl,
		writer:  writer,
		timeout: periodInit,
		stop:    make(chan bool),
	}
}

func (ie *ifaceBwEntry) ProcessPingResults(pr []pinger.PingResult) {
	for _, pingres := range pr {
		entry := peerDataEntry{
			IP:      pingres.IP,
			Latency: pingres.Latency,
			Loss:    pingres.Loss,
		}

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

		ie.Peers = append(ie.Peers, entry)
	}
	ie.channel <- ie
}

func (wpw *WgPeerWatcher) execute() error {
	wg := wpw.wg
	resp := peerBwData{}
	resp.ID = "UNKNOWN"
	resp.MsgType = "IFACES_PEERS_BW_DATA"

	wgdevs, err := wg.Devices()
	if err != nil {
		return err
	}

	count := len(wgdevs)

	// If no interfaces are created yet - I send nothing to controller and wait a short time
	// When interfaces are created - switch to less frequently check
	if count == 0 {
		if wpw.timeout != periodInit {
			wpw.timeout = periodInit
			wpw.ticker.Reset(wpw.timeout)
		}
		return nil
	} else if wpw.timeout != periodRun {
		wpw.timeout = periodRun
		wpw.ticker.Reset(wpw.timeout)
	}

	// The pinger runs in background, so I will create a channel with number of interfaces
	// And pinger callback will put interface entry (with peers ping info) into channel
	c := make(chan *ifaceBwEntry, count)

	for _, wgdev := range wgdevs {
		ifaceData := ifaceBwEntry{
			IfName:    wgdev.Name,
			PublicKey: wgdev.PublicKey.String(),
			Peers:     []peerDataEntry{},
			channel:   c,
		}
		ping := pinger.NewPinger(&ifaceData)

		for _, p := range wgdev.Peers {
			if len(p.AllowedIPs) == 0 {
				continue
			}
			ip := p.AllowedIPs[0]
			ping.AddHost(ip.IP.String())
		}

		ping.RunOnce()
	}

	for count > 0 {
		entry := <-c
		resp.Data = append(resp.Data, *entry)
		count--
	}
	close(c)

	resp.Now()
	raw, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	wpw.writer.Write(raw)

	return nil
}

func (wpw *WgPeerWatcher) Start() {
	// I'm not doing concurency prevention here,
	// because this Start() should not be called concurently and this is only a sanity check
	if wpw.ticker != nil {
		log.Fatal("A peer watcher is already running")
	}

	wpw.ticker = time.NewTicker(periodInit)
	go func() {
		for {
			select {
			case <-wpw.stop:
				return
			case <-wpw.ticker.C:
				wpw.execute()

			}
		}
	}()
}

func (wpw *WgPeerWatcher) Stop() {
	// Cannot stop not running instance
	if wpw.ticker == nil {
		return
	}

	wpw.ticker.Stop()
	wpw.stop <- true
}
