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

type wgRouteMsg struct {
	common.MessageHeader
	Data []wgRouteEntry `json:"data"`
}

func NewMsg() *wgRouteMsg {
	msg := wgRouteMsg{
		Data: []wgRouteEntry{},
	}
	msg.MsgType = cmd
	msg.ID = env.MessageDefaultID

	return &msg
}

func (msg *wgRouteMsg) Send(w io.Writer) error {
	if len(msg.Data) == 0 {
		return nil
	}

	msg.Now()
	raw, err := json.Marshal(msg)
	if err != nil {
		logger.Error().Println(pkgName, "json", err)
		return err
	}

	_, err = w.Write(raw)
	return err
}

func (msg *wgRouteMsg) Add(rrs []common.RouteResult) error {
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
		msg.Data = append(msg.Data, re)
	}

	return nil
}
