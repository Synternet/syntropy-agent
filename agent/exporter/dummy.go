package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
)

type DummyCollector struct {
}

func (dc *DummyCollector) Describe(ch chan<- *prometheus.Desc) {
}

func (dc *DummyCollector) Collect(ch chan<- prometheus.Metric) {
}
