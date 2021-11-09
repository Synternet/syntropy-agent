package ipadmsg

import (
	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
)

// TODO: not yet sure is it too much honour for this message to live in separate package
// or it should be merged with router/servicemon
const Cmd = "IFACES_PEERS_ACTIVE_DATA"

type PeerActiveDataEntry struct {
	PreviousConnID int    `json:"prev_connection_id"`
	ConnectionID   int    `json:"connection_id"`
	GroupID        int    `json:"connection_group_id"`
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
	resp.MsgType = Cmd
	return &resp
}
