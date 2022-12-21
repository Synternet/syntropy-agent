// router package is used to setup routes to peers and services
// Router uses PeerMonitod which actively monitores direct and (sdn) wireguard peers
// And ServiceMonitor for changing routes to configured services
package router

import (
	"io"
	"sync"

	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/routeselector"
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
	pmCfg  routeselector.RouteSelectorConfig
}

func New(w io.Writer) *Router {
	diff, ratio := config.RerouteThresholds()
	return &Router{
		writer: w,
		routes: make(map[int]*routerGroupEntry),
		pmCfg: routeselector.RouteSelectorConfig{
			AverageSize:              config.PeerCheckWindow(),
			RouteStrategy:            config.GetRouteStrategy(),
			RerouteRatio:             ratio,
			RerouteDiff:              diff,
			RouteDeleteLossThreshold: float32(config.GetRouteDeleteThreshold()),
		},
	}
}
