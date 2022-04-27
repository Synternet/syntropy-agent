// router package is used to setup routes
// also actively monitores direct and (sdn) wireguard peers
// and setups best routing path
package router

import (
	"io"
	"sync"

	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
)

const (
	pkgName = "Router. "
	cmd     = "SMART_ROUTER"
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

func (obj *Router) execute() {
	resp := peeradata.NewMessage()

	for _, routeGroup := range obj.routes {
		rv := routeGroup.serviceMonitor.Reroute(routeGroup.peerMonitor.BestPath())
		if rv != nil {
			resp.Add(rv)
		}
	}

	resp.Send(obj.writer)
}
