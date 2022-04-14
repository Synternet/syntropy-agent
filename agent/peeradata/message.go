package peeradata

import (
	"encoding/json"
	"io"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const (
	cmd     = "IFACES_PEERS_ACTIVE_DATA"
	pkgName = "PeersActiveData. "
)

type Message struct {
	common.MessageHeader
	Data []*Entry `json:"data"`
}

func NewMessage() *Message {
	resp := &Message{
		Data: []*Entry{},
	}
	resp.ID = env.MessageDefaultID
	resp.MsgType = cmd
	return resp
}

func (msg *Message) Add(entries ...*Entry) {
	// Agent is treating each service route separately
	// But controler is expecting all services in once connection group
	// (identified by the same connection_group_id) to be treated the same.
	// Thus I need to to send only one unique route change entry per connection group

	unique := func(list []*Entry, newEntry *Entry) bool {
		for _, curr := range list {
			if curr.GroupID == newEntry.GroupID {
				return false
			}
		}
		return true
	}

	for _, entry := range entries {
		if unique(msg.Data, entry) {
			msg.Data = append(msg.Data, entry)
		}
	}
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
