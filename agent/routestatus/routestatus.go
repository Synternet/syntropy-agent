package routestatus

import (
	"encoding/json"
	"io"

	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
)

const (
	cmd         = "WG_ROUTE_STATUS"
	pkgName     = "WgRouteStatus. "
	statusOK    = "OK"
	statusError = "ERROR"
)

type wgRouteEntry struct {
	Status  string `json:"status"`
	IP      string `json:"ip"`
	Message string `json:"msg,omitempty"`
}

type wgConnectionEntry struct {
	ConnectionID int            `json:"connection_id,omitempty"`
	GroupID      int            `json:"connection_group_id,omitempty"`
	RouteStatus  []wgRouteEntry `json:"statuses"`
}

type wgRouteStatusMsg struct {
	common.MessageHeader
	Data []wgConnectionEntry `json:"data"`
}

func NewMsg() *wgRouteStatusMsg {
	msg := wgRouteStatusMsg{
		Data: []wgConnectionEntry{},
	}
	msg.MsgType = cmd
	msg.ID = env.MessageDefaultID

	return &msg
}

func (msg *wgRouteStatusMsg) Send(w io.Writer) error {
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

func (msg *wgRouteStatusMsg) Add(connID, grID int, rrs []common.RouteResult) error {
	ce := wgConnectionEntry{
		ConnectionID: connID,
		GroupID:      grID,
		RouteStatus:  []wgRouteEntry{},
	}

	for _, rres := range rrs {
		re := wgRouteEntry{
			IP: rres.IP,
		}
		if rres.Error == nil {
			re.Status = statusOK
		} else {
			re.Status = statusError
			re.Message = rres.Error.Error()
		}
		ce.RouteStatus = append(ce.RouteStatus, re)
	}
	msg.Data = append(msg.Data, ce)

	return nil
}
