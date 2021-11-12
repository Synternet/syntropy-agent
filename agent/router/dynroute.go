// router package is used to setup routes
// also actively monitores direct and (sdn) wireguard peers
// and setups best routing path
package router

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/agent/router/peermon"
	"github.com/SyntropyNet/syntropy-agent-go/agent/router/servicemon"
)

const (
	checkPeriod = time.Second * 3
	pkgName     = "Router. "
	cmd         = "SMART_ROUTER"
)

type Router struct {
	sync.Mutex
	writer         io.Writer
	peerMonitor    *peermon.PeerMonitor
	serviceMonitor *servicemon.ServiceMonitor
}

func New(w io.Writer) *Router {
	return &Router{
		writer:         w,
		peerMonitor:    &peermon.PeerMonitor{},
		serviceMonitor: servicemon.New(w),
	}
}

func (obj *Router) Name() string {
	return cmd
}

func (obj *Router) execute() {
	obj.serviceMonitor.Reroute(obj.peerMonitor.BestPath())
}

func (obj *Router) Run(ctx context.Context) error {
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
