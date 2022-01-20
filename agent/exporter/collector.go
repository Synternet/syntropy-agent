package exporter

import (
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector interface {
	multiping.PingClient
	AddPeer(ip, ifname, pubkey string, connID, grID int)
	Describe(ch chan<- *prometheus.Desc)
	Collect(ch chan<- prometheus.Metric)
}
