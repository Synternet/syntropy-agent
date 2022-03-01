package exporter

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	pkgName = "PrometheusExporter. "
	cmd     = "EXPORTER"
)

type PeersMetrics struct {
	port uint16
	reg  *prometheus.Registry
}

func New(port uint16, collector prometheus.Collector) (*PeersMetrics, error) {
	obj := PeersMetrics{
		port: port,
		reg:  prometheus.NewRegistry(),
	}

	err := obj.reg.Register(collector)
	if err != nil {
		return nil, err
	}

	return &obj, nil
}

func (obj *PeersMetrics) Run(ctx context.Context) error {
	handler := promhttp.HandlerFor(obj.reg, promhttp.HandlerOpts{})
	mux := http.NewServeMux()
	mux.Handle("/metrics", handler)

	logger.Debug().Println(pkgName, "exporter starting on port", obj.port)
	srv := http.Server{
		Addr:         fmt.Sprintf(":%d", obj.port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			logger.Error().Println(pkgName, err)
		}
	}()

	go func() {
		<-ctx.Done()
		logger.Debug().Println(pkgName, "stopping", cmd)
		srv.Close()
	}()

	return nil
}

func (obj *PeersMetrics) Name() string {
	return cmd
}
