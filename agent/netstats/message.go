package netstats

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
)

const (
	cmd      = "IFACES_PEERS_BW_DATA"
	pkgName  = "Peer_Data. "
	PingLoss = 1.0
)

type PeerDataEntry struct {
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

type IfaceBwEntry struct {
	IfName    string           `json:"iface"`
	PublicKey string           `json:"iface_public_key"`
	Peers     []*PeerDataEntry `json:"peers"`
}

type Message struct {
	common.MessageHeader
	Data []IfaceBwEntry `json:"data"`
}

func NewMessage() *Message {
	msg := &Message{
		Data: []IfaceBwEntry{},
	}
	msg.ID = env.MessageDefaultID
	msg.MsgType = cmd

	return msg
}

func (msg *Message) Send(writer io.Writer) error {
	if len(msg.Data) == 0 {
		// no need send an empty message
		return nil
	}

	msg.Now()
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	logger.Debug().Println(pkgName, "Sending: ", string(raw))
	_, err = writer.Write(raw)
	return err
}

// Parse ping result and fill statistics for connected peers
func (msg *Message) PingProcess(pr *multiping.PingData) {
	for _, ifaceEntry := range msg.Data {
		for _, peerEntry := range ifaceEntry.Peers {
			val, ok := pr.Get(peerEntry.IP)
			if ok {
				// format results for controler
				peerEntry.Latency = val.Latency()
				peerEntry.Loss = val.Loss()
			}
		}
	}
}

// Add a single peer statistics to a message
func (msg *Message) Add(ip string, latency, loss float32) error {
	for _, ifaceEntry := range msg.Data {
		for _, peerEntry := range ifaceEntry.Peers {
			if peerEntry.IP == ip {
				peerEntry.Latency = latency
				peerEntry.Loss = loss
				return nil
			}
		}
	}
	return fmt.Errorf("ip %s not found", ip)
}
