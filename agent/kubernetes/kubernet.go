package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/google/go-cmp/cmp"
)

const (
	pkgName = "Kubernetes. "
	cmd     = "KUBERNETES_SERVICE_INFO"
)

type kubernet struct {
	writer     io.Writer
	httpClient *http.Client
	baseURL    string
	namespaces []string
	msg        kubernetesInfoMessage
	ctx        context.Context
}

func New(w io.Writer) common.Service {
	kub := kubernet{
		writer: w,
	}
	kub.msg.MsgType = cmd
	kub.msg.ID = env.MessageDefaultID

	return &kub
}

func (obj *kubernet) Name() string {
	return cmd
}

func (obj *kubernet) execute() {
	services, err := obj.monitorServices()

	if err != nil {
		// If error occurred agent should not send empty services list,
		// because it will be treated as valid message.
		logger.Warning().Println(pkgName, "listing services", err)
		return
	}

	if !cmp.Equal(services, obj.msg.Data) {
		obj.msg.Data = services
		obj.msg.Now()
		raw, err := json.Marshal(obj.msg)
		if err != nil {
			logger.Error().Println(pkgName, "json marshal", err)
			return
		}
		logger.Debug().Println(pkgName, "Sending: ", string(raw))
		obj.writer.Write(raw)
	}
}

func (obj *kubernet) Run(ctx context.Context) error {
	if obj.ctx != nil {
		return fmt.Errorf("kubernetes watcher already running")
	}
	obj.ctx = ctx

	err := obj.initClient()
	if err != nil {
		logger.Error().Println(pkgName, err)
		return err
	}

	go func() {
		ticker := time.NewTicker(config.PeerCheckTime())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Debug().Println(pkgName, "stopping", cmd)
				obj.httpClient.CloseIdleConnections()
				obj.httpClient = nil
				return
			case <-ticker.C:
				obj.execute()
			}
		}
	}()
	return nil
}
