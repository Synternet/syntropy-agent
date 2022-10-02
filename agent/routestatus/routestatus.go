package routestatus

import (
	"encoding/json"
	"io"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const (
	cmd     = "SERVICE_STATUS"
	pkgName = "ServiceStatus. "
)

type Message struct {
	common.MessageHeader
	Data []*Connection `json:"data"`
}

func New() *Message {
	msg := Message{
		Data: []*Connection{},
	}
	msg.MsgType = cmd
	msg.ID = env.MessageDefaultID

	return &msg
}

// Send message to controller (writer)
func (msg *Message) Send(w io.Writer) error {
	if len(msg.Data) == 0 {
		return nil
	}

	msg.Now()
	raw, err := json.Marshal(msg)
	if err != nil {
		logger.Error().Println(pkgName, "json", err)
		return err
	}

	logger.Debug().Println(pkgName, "Sending: ", string(raw))
	_, err = w.Write(raw)
	return err
}

// Adds new connections to array.
// Performs smart merge, if connections have same IDs
func (msg *Message) Add(connections ...*Connection) {
	for _, newConn := range connections {
		merged := false
		for _, conn := range msg.Data {
			if conn.ConnectionID == newConn.ConnectionID && conn.GroupID == newConn.GroupID {
				conn.RouteStatus = append(conn.RouteStatus, newConn.RouteStatus...)
				merged = true
			}
		}
		if !merged {
			msg.Data = append(msg.Data, newConn)
		}
	}
}
