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
	msg.Data = append(msg.Data, entries...)
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
