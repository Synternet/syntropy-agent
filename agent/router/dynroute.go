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
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
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

func (r *Router) rerouteServices() {
	resp := peeradata.NewMessage()
	count := 0

	for _, routeGroup := range r.routes {
		// Change routes to configured services
		// and build a message to controler
		rv := routeGroup.serviceMonitor.Reroute(routeGroup.peerMonitor.BestPath())
		if rv != nil {
			resp.Add(rv)
			count++
		}
	}

	resp.Send(r.writer)
	if count > 0 {
		logger.Info().Println(pkgName, "Rerouted services for", count, "connections")
	}
}
