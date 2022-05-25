// autoping package implement both: controller.Command and controller.Service
package autoping

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
)

const (
	cmd     = "AUTO_PING"
	pkgName = "Auto_Ping. "
)

type AutoPing struct {
	sync.RWMutex
	ctx      context.Context
	writer   io.Writer
	pinger   *multiping.MultiPing
	pingData *multiping.PingData
	timer    *time.Ticker
	results  []byte
}

func New(w io.Writer, p *multiping.MultiPing) *AutoPing {
	ap := AutoPing{
		writer:   w,
		pinger:   p,
		pingData: multiping.NewPingData(),
	}

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
	obj.pingData.Flush()
	for _, ipstr := range req.Data.IPs {
		ip, err := netip.ParseAddr(ipstr)
		if err != nil {
			logger.Warning().Println(pkgName, "invalid address", ipstr, err)
			continue
		}

		obj.pingData.Add(ip)
	}

	if obj.pingData.Count() > 0 {
		obj.start(time.Duration(req.Data.Interval) * time.Second)
	}

	return nil
}

func (obj *AutoPing) PingProcess(pr *multiping.PingData) {
	resp := newResponceMsg()

	resp.PingProcess(pr)

	// Clear old statistics
	pr.Reset()

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
	if obj.ctx == nil {
		logger.Error().Println(pkgName, "service is not started")
		return
	}

	obj.timer = time.NewTicker(period)
	go func() {
		// Don't wait for ticker and do the first ping asap
		obj.pinger.Ping(obj.pingData)
		obj.PingProcess(obj.pingData)

		defer obj.timer.Stop()
		for {
			select {
			case <-obj.ctx.Done():
				logger.Debug().Println(pkgName, "stopping", cmd)
				return
			case <-obj.timer.C:
				obj.pinger.Ping(obj.pingData)
				obj.PingProcess(obj.pingData)
			}
		}
	}()
}

func (obj *AutoPing) Run(ctx context.Context) error {
	if obj.ctx != nil {
		return fmt.Errorf("%s is already running", pkgName)
	}
	obj.ctx = ctx

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
