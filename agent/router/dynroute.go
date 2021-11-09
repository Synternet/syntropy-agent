// router package is used to setup routes
// also actively monitores direct and (sdn) wireguard peers
// and setups best routing path
package router

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/agent/router/ipadmsg"
	"github.com/SyntropyNet/syntropy-agent-go/agent/router/peermon"
	"github.com/SyntropyNet/syntropy-agent-go/agent/router/servicemon"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/scontext"
)

const (
	checkPeriod = time.Second * 3
	pkgName     = "Router. "
)

type Router struct {
	sync.Mutex
	writer         io.Writer
	peerMonitor    *peermon.PeerMonitor
	serviceMonitor *servicemon.ServiceMonitor
	ctx            scontext.StartStopContext
}

func New(ctx context.Context, w io.Writer) *Router {
	return &Router{
		writer:         w,
		peerMonitor:    &peermon.PeerMonitor{},
		serviceMonitor: servicemon.New(w),
		ctx:            scontext.New(ctx),
	}
}

func (obj *Router) Name() string {
	return ipadmsg.Cmd
}

func (obj *Router) execute() {
	obj.serviceMonitor.Reroute(obj.peerMonitor.BestPath())
}

func (obj *Router) Start() error {
	ctx, err := obj.ctx.CreateContext()
	if err != nil {
		return fmt.Errorf("dynamic routing already running")
	}

	go func() {
		ticker := time.NewTicker(checkPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				obj.execute()

			}
		}
	}()

	return nil
}

func (obj *Router) Stop() error {
	if err := obj.ctx.CancelContext(); err != nil {
		return fmt.Errorf("dynamic routing is not running")
	}

	return nil
}
