package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	pkgName = "Metrics. "
	cmd     = "PROMETHEUS"
)

type peersMetrics struct {
	port      uint16
	collector peersCollector
	reg       *prometheus.Registry
}

func New(port uint16) (common.Service, error) {
	logger.Debug().Println(pkgName, "Metrics exporter enabled on port", port)
	obj := peersMetrics{
		port: port,
	}
	obj.reg = prometheus.NewRegistry()
	err := obj.reg.Register(obj.collector)
	if err != nil {
		return nil, err
	}

	return &obj, nil
}

func (obj *peersMetrics) Run(ctx context.Context) error {
	handler := promhttp.HandlerFor(obj.reg, promhttp.HandlerOpts{})
	mux := http.NewServeMux()
	mux.Handle("/metrics", handler)

	srv := http.Server{
		Addr:         fmt.Sprintf(":%d", obj.port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go srv.ListenAndServe()

	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	return nil
}

func (obj *peersMetrics) Name() string {
	return cmd
}
