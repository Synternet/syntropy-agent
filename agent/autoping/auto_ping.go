// autoping package implement both: controller.Command and controller.Service
package autoping

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/scontext"
)

const (
	cmd     = "AUTO_PING"
	pkgName = "Auto_Ping. "
)

type AutoPing struct {
	sync.RWMutex
	// TODO: no worries, this mess will be fixed ASAP
	ctx     scontext.StartStopContext
	ctx2    context.Context
	writer  io.Writer
	ping    *multiping.MultiPing
	timer   *time.Ticker
	results []byte
}

func New(ctx context.Context, w io.Writer) *AutoPing {
	ap := AutoPing{
		writer: w,
		ctx:    scontext.New(ctx),
	}
	ap.ping = multiping.New(&ap)
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

	obj.Lock()
	defer obj.Unlock()

	obj.stop()
	obj.ping.Flush()
	obj.ping.AddHost(req.Data.IPs...)
	if obj.ping.Count() > 0 {
		obj.start(time.Duration(req.Data.Interval) * time.Second)
	}

	return nil
}

func (obj *AutoPing) PingProcess(pr *multiping.PingResult) {
	resp := newResponceMsg()

	// TODO: respect controllers set LimitCount
	pr.Iterate(func(ip string, val multiping.PingResultValue) {
		resp.Data.Pings = append(resp.Data.Pings,
			pingResponseEntry{
				IP:      ip,
				Latency: val.Latency,
				Loss:    val.Loss,
			})
	})

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

func (obj *AutoPing) stop() {
	if obj.timer != nil {
		obj.timer.Stop()
		obj.timer = nil
	}
}

func (obj *AutoPing) start(period time.Duration) {
	obj.timer = time.NewTicker(period)
	go func() {
		defer obj.timer.Stop()
		for {
			select {
			case <-obj.ctx2.Done():
				return
			case <-obj.timer.C:
				obj.ping.Ping()
			}
		}
	}()
}

func (obj *AutoPing) Start() error {
	var err error
	obj.ctx2, err = obj.ctx.CreateContext()
	if err != nil {
		return fmt.Errorf("%s is already running", pkgName)
	}
	return nil
}

func (obj *AutoPing) Stop() error {
	if err := obj.ctx.CancelContext(); err != nil {
		return fmt.Errorf("auto_ping is not running")
	}

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
