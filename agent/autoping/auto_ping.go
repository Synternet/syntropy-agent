// autoping package implement both: controller.Command and controller.Service
package autoping

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
)

const cmd = "AUTO_PING"
const pkgName = "Auto_Ping. "

type AutoPing struct {
	sync.RWMutex
	writer  io.Writer
	ping    *multiping.MultiPing
	results []byte
}

type autoPingRequest struct {
	common.MessageHeader
	Data struct {
		IPs       []string `json:"ips"`
		Interval  int      `json:"interval"`
		RespLimit int      `json:"response_limit"`
	} `json:"data"`
}

type autoPingResponse struct {
	common.MessageHeader
	Data struct {
		Pings []multiping.PingResult `json:"pings"`
	} `json:"data"`
}

func New(ctx context.Context, w io.Writer) *AutoPing {
	ap := AutoPing{
		writer: w,
	}
	ap.ping = multiping.New(ctx, &ap)
	return &ap
}

func (obj *AutoPing) Name() string {
	return cmd
}

func (obj *AutoPing) Exec(raw []byte) error {

	var req autoPingRequest
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	obj.ping.Stop()
	obj.ping.Period = time.Duration(req.Data.Interval) * time.Second
	obj.ping.LimitCount = req.Data.RespLimit
	obj.ping.Flush()
	obj.ping.AddHost(req.Data.IPs...)
	obj.ping.Start()

	return nil
}

func (obj *AutoPing) PingProcess(pr []multiping.PingResult) {
	var resp autoPingResponse
	resp.Data.Pings = pr
	resp.MsgType = cmd
	resp.ID = env.MessageDefaultID
	resp.Now()

	if len(resp.Data.Pings) > 0 {
		var err error
		obj.Lock()
		obj.results, err = json.Marshal(resp)
		obj.Unlock()
		if err != nil {
			logger.Error().Println(pkgName, "Process Ping Results: ", err)
			return
		}

		obj.RLock()
		obj.writer.Write(obj.results)
		obj.RUnlock()
	}
}

func (obj *AutoPing) Start() error {
	obj.ping.Start()
	return nil
}

func (obj *AutoPing) Stop() error {
	obj.ping.Stop()
	return nil
}

func (obj *AutoPing) SupportInfo() *common.KeyValue {
	obj.RLock()
	defer obj.RUnlock()

	return &common.KeyValue{
		Key:   cmd,
		Value: string(obj.results),
	}
}
