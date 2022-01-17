package metrics

import (
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
)

type peersCollector struct {
}

func (pc peersCollector) Describe(ch chan<- *prometheus.Desc) {
	logger.Debug().Println(pkgName, "Describe")
}

func (pc peersCollector) Collect(ch chan<- prometheus.Metric) {
	logger.Debug().Println(pkgName, "Collect")
}
