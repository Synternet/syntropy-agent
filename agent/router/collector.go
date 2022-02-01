package router

import (
	"github.com/prometheus/client_golang/prometheus"
)

func (r *Router) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(r, ch)
}

func (r *Router) Collect(ch chan<- prometheus.Metric) {
	r.Lock()
	defer r.Unlock()

	for groupID, route := range r.routes {
		route.peerMonitor.Collect(ch, groupID)
	}
}
