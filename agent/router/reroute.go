package router

import (
	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
)

func (r *Router) PingProcess(pr *multiping.PingData) {
	r.Lock()
	defer r.Unlock()

	for _, pm := range r.routes {
		pm.peerMonitor.PingProcess(pr)
	}
	// After processing ping results check for a better route for services
	r.rerouteServices()
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
