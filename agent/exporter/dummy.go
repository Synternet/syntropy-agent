package exporter

import (
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
	"github.com/prometheus/client_golang/prometheus"
)

type DummyCollector struct {
}

func (dc *DummyCollector) AddPeer(ip, ifname, pubkey string, connID, grID int) {
}

func (dc *DummyCollector) PingProcess(pr *multiping.PingData) {
}

func (dc *DummyCollector) Describe(ch chan<- *prometheus.Desc) {
}

func (dc *DummyCollector) Collect(ch chan<- prometheus.Metric) {
}
