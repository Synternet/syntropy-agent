package router

import "github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"

func (r *Router) PingProcess(pr []multiping.PingResult) {
	r.peerMonitor.PingProcess(pr)
}
