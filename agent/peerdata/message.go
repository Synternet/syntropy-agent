package peerdata

import (
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
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
	RxBytes      int64   `json:"rx_bytes"`
	TxBytes      int64   `json:"tx_bytes"`
	RxSpeed      float32 `json:"rx_speed_mbps"`
	TxSpeed      float32 `json:"tx_speed_mbps"`
	ConnectionID int     `json:"connection_id"`
	GroupID      int     `json:"connection_group_id"`
}

type ifaceBwEntry struct {
	IfName    string           `json:"iface"`
	PublicKey string           `json:"iface_public_key"`
	Peers     []*peerDataEntry `json:"peers"`
}

type peerBwData struct {
	common.MessageHeader
	Data []ifaceBwEntry `json:"data"`
}

func newMsg() *peerBwData {
	return &peerBwData{
		Data: []ifaceBwEntry{},
	}
}

// Parse ping result and fill statistics for connected peers
func (msg *peerBwData) PingProcess(pr *multiping.PingData) {
	for _, ifaceEntry := range msg.Data {
		for _, peerEntry := range ifaceEntry.Peers {
			val, ok := pr.Get(peerEntry.IP)
			if !ok {
				logger.Warning().Println(pkgName, peerEntry.IP, "missing in ping results")
				continue
			}

			// format results for controler
			peerEntry.Latency = val.Latency()
			peerEntry.Loss = val.Loss()
		}
	}
}
