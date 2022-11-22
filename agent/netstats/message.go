package netstats

import (
	"encoding/json"
	"fmt"
	"io"
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping/pingdata"
)

const (
	cmd      = "NETWORK_STATISTICS"
	pkgName  = "Peer_Data. "
	PingLoss = 1.0
)

type PeerDataEntry struct {
	ConnectionID int     `json:"connection_id"`
	GroupID      int     `json:"connection_group_id"`
	Handshake    string  `json:"last_handshake,omitempty"`
	Latency      float32 `json:"latency_ms"`
	Loss         float32 `json:"packet_loss"`
	RxBytes      int64   `json:"rx_bytes"`
	TxBytes      int64   `json:"tx_bytes"`
	RxSpeed      float32 `json:"rx_bps"`
	TxSpeed      float32 `json:"tx_bps"`
	IP           string  `json:"internal_ip,omitempty"`
}

type IfaceBwEntry struct {
	IfIndex int              `json:"index"`
	Peers   []*PeerDataEntry `json:"peers"`
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

	logger.Message().Println(pkgName, "Sending: ", string(raw))
	_, err = writer.Write(raw)
	return err
}

// Parse ping result and fill statistics for connected peers
func (msg *Message) PingProcess(pr *pingdata.PingData) {
	for _, ifaceEntry := range msg.Data {
		for _, peerEntry := range ifaceEntry.Peers {
			addr, err := netip.ParseAddr(peerEntry.IP)
			if err != nil {
				continue
			}
			val, ok := pr.Get(addr)
			if ok {
				if val.Valid() {
					// format results for controler
					peerEntry.Latency = val.Latency()
					peerEntry.Loss = val.Loss()
				} else {
					logger.Warning().Println(pkgName, "Invalid ping stats for", addr)
				}
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
