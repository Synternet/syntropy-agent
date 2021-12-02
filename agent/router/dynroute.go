// router package is used to setup routes
// also actively monitores direct and (sdn) wireguard peers
// and setups best routing path
package router

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const (
	checkPeriod = time.Second * 3
	pkgName     = "Router. "
	cmd         = "SMART_ROUTER"
)

type Router struct {
	sync.Mutex
	writer io.Writer
	routes map[int]*routerGroupEntry // route list ordered by group_id
}

func New(w io.Writer) *Router {
	return &Router{
		writer: w,
		routes: make(map[int]*routerGroupEntry),
	}
}

func (obj *Router) Name() string {
	return cmd
}

func (obj *Router) execute() {
	resp := peeradata.NewMessage()

	for _, routeGroup := range obj.routes {
		rv := routeGroup.serviceMonitor.Reroute(routeGroup.peerMonitor.BestPath())
		resp.Data = append(resp.Data, rv...)
	}

	if len(resp.Data) > 0 {
		resp.Now()
		raw, err := json.Marshal(resp)
		if err != nil {
			logger.Error().Println(pkgName, err)
			return
		}

		logger.Debug().Println(pkgName, "Sending: ", string(raw))
		obj.writer.Write(raw)
	}

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
