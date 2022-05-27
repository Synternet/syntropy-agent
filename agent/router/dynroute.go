// router package is used to setup routes
// also actively monitores direct and (sdn) wireguard peers
// and setups best routing path
package router

import (
	"io"
	"sync"

	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
)

const (
	pkgName = "Router. "
	cmd     = "SMART_ROUTER"
)

type Router struct {
	sync.Mutex
	writer io.Writer
	routes map[int]*routerGroupEntry // route list ordered by group_id
	pmCfg  peermon.PeerMonitorConfig
}

func New(w io.Writer) *Router {
	diff, ratio := config.RerouteThresholds()
	return &Router{
		writer: w,
		routes: make(map[int]*routerGroupEntry),
		pmCfg: peermon.PeerMonitorConfig{
			AverageSize:              config.PeerCheckWindow(),
			RouteStrategy:            config.GetRouteStrategy(),
			RerouteRatio:             ratio,
			RerouteDiff:              diff,
			RouteDeleteLossThreshold: float32(config.GetRouteDeleteThreshold()),
		},
	}
}

func (obj *Router) execute() {
	resp := peeradata.NewMessage()

	for _, routeGroup := range obj.routes {
		// Nothing to do on when no services are configured
		if routeGroup.serviceMonitor.Count() == 0 {
			continue
		}

		// Change routes to configured services
		// and build a message to controler
		rv := routeGroup.serviceMonitor.Reroute(routeGroup.peerMonitor.BestPath())
		if rv != nil {
			resp.Add(rv)
		}
	}

	resp.Send(obj.writer)
}
