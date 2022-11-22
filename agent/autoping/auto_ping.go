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
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping/pingdata"
)

const (
	cmd     = "AUTO_PING"
	pkgName = "Auto_Ping. "
)

type AutoPing struct {
	sync.Mutex
	ctx      context.Context
	writer   io.Writer
	pinger   *multiping.MultiPing
	pingData *pingdata.PingData
	timer    *time.Ticker
	results  []byte
}

func New(w io.Writer, p *multiping.MultiPing) *AutoPing {
	ap := AutoPing{
		writer:   w,
		pinger:   p,
		pingData: pingdata.NewPingData(),
		timer:    time.NewTicker(time.Second),
	}

	// Stop ticker so no ticks will be scheduled
	// Will reset timer on demand later
	ap.timer.Stop()

	return &ap
}

func (obj *AutoPing) Name() string {
	return cmd
}

func (obj *AutoPing) Exec(req common.ConfigSettingsAutopingEntry) error {

	obj.Lock()
	defer obj.Unlock()

	// stop the timer
	obj.timer.Stop()

	// set new autoping data
	obj.pingData.Flush()
	for _, ipstr := range req.IPs {
		ip, err := netip.ParseAddr(ipstr)
		if err != nil {
			logger.Warning().Println(pkgName, "invalid address", ipstr, err)
			continue
		}
		obj.pingData.Add(ip)
	}

	// Reschedule ping ticker
	if obj.pingData.Count() > 0 && req.Interval > 0 {
		obj.timer.Reset(time.Duration(req.Interval) * time.Second)
		// We want first ping result asap, so first iteration is now.
		// PingAction uses a lock inside, thus I do this in separate goroutine,
		// after the unlocked in defer
		go obj.pingAction()
	}

	return nil
}

func (obj *AutoPing) pingAction() {
	obj.Lock()
	defer obj.Unlock()
	if obj.pingData.Count() <= 0 {
		return
	}

	// perform pinging
	obj.pinger.Ping(obj.pingData)

	// Process results
	resp := newResponceMsg()
	resp.PingProcess(obj.pingData)

	// Clear old statistics
	obj.pingData.Reset()

	// marshal and report results
	var err error
	obj.results, err = json.Marshal(resp)
	if err != nil {
		logger.Error().Println(pkgName, "Process Ping Results: ", err)
	} else {
		obj.writer.Write(obj.results)
	}
}

func (obj *AutoPing) Run(ctx context.Context) error {
	if obj.ctx != nil {
		return fmt.Errorf("%s is already running", pkgName)
	}
	obj.ctx = ctx

	go func() {
		for {
			select {
			case <-obj.ctx.Done():
				logger.Debug().Println(pkgName, "stopping", cmd)
				obj.timer.Stop()
				obj.pingData.Flush()
				return
			case <-obj.timer.C:
				obj.pingAction()
			}
		}
	}()

	return nil
}

func (obj *AutoPing) SupportInfo() *common.KeyValue {
	obj.Lock()
	defer obj.Unlock()

	return &common.KeyValue{
		Key:   cmd,
		Value: string(obj.results),
	}
}
