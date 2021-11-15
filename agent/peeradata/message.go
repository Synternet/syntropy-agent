package peeradata

import (
	"encoding/json"
	"io"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
)

const (
	cmd     = "IFACES_PEERS_ACTIVE_DATA"
	pkgName = "PeersActiveData. "
)

type PeerActiveDataEntry struct {
	PreviousConnID int    `json:"prev_connection_id,omitempty"`
	ConnectionID   int    `json:"connection_id"`
	GroupID        int    `json:"connection_group_id,omitempty"`
	Timestamp      string `json:"timestamp"`
}

type PeersActiveDataMessage struct {
	common.MessageHeader
	Data []PeerActiveDataEntry `json:"data"`
}

func NewMessage() *PeersActiveDataMessage {
	resp := PeersActiveDataMessage{
		Data: []PeerActiveDataEntry{},
	}
	resp.ID = env.MessageDefaultID
	resp.MsgType = cmd
	return &resp
}

func (pad *PeersActiveDataMessage) Add(entries ...PeerActiveDataEntry) {
	pad.Data = append(pad.Data, entries...)
}

func (pad *PeersActiveDataMessage) Send(writer io.Writer) error {
	if len(pad.Data) == 0 {
		// controler does not need an empty message
		return nil
	}

	pad.Now()
	raw, err := json.Marshal(pad)
	if err != nil {
		return err
	}

	logger.Debug().Println(pkgName, "Sending: ", string(raw))
	_, err = writer.Write(raw)
	return err
}
