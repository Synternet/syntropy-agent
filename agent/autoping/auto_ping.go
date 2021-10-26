// autoping package implement both: controller.Command and controller.Service
package autoping

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/slock"
)

const cmd = "AUTO_PING"
const pkgName = "Auto_Ping. "

type autoPing struct {
	slock.AtomicServiceLock
	writer io.Writer
	ping   *multiping.MultiPing
}

type autoPingRequest struct {
	common.MessageHeader
	Data struct {
		IPs       []string `json:"ips"`
		Interval  int      `json:"interval"`
		RespLimit int      `json:"responce_limit"`
	} `json:"data"`
}

type autoPingResponce struct {
	common.MessageHeader
	Data struct {
		Pings []multiping.PingResult `json:"pings"`
	} `json:"data"`
}

func New(ctx context.Context, w io.Writer) common.CommandService {
	ap := autoPing{
		writer: w,
	}
	ap.ping = multiping.New(ctx, &ap)
	return &ap
}

func (obj *autoPing) Name() string {
	return cmd
}

func (obj *autoPing) Exec(raw []byte) error {

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

func (obj *autoPing) PingProcess(pr []multiping.PingResult) {
	var resp autoPingResponce
	resp.Data.Pings = pr
	resp.MsgType = cmd
	resp.ID = env.MessageDefaultID
	resp.Now()

	if len(resp.Data.Pings) > 0 {
		arr, err := json.Marshal(resp)
		if err != nil {
			logger.Error().Println(pkgName, "Process Ping Results: ", err)
			return
		}

		obj.writer.Write(arr)
	}
}

func (obj *autoPing) Start() error {
	if !obj.TryLock() {
		return fmt.Errorf("%s is already started", pkgName)
	}
	obj.ping.Start()
	return nil
}

func (obj *autoPing) Stop() error {
	if !obj.TryUnlock() {
		return fmt.Errorf("%s is not running", pkgName)
	}

	obj.ping.Stop()
	return nil
}
