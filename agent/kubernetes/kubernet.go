package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/google/go-cmp/cmp"
	"k8s.io/client-go/kubernetes"
)

// TODO (later): in future think about optimising binary size
// and using GO stdlib kubernetes package
// (premature optimisation is the root of all evil)
const (
	pkgName      = "Kubernetes. "
	cmd          = "KUBERNETES_SERVICE_INFO"
	updatePeriod = time.Second * 5
)

type kubernet struct {
	writer io.Writer
	klient *kubernetes.Clientset
	msg    kubernetesInfoMessage
	ctx    context.Context
}

func New(w io.Writer) common.Service {
	kub := kubernet{
		writer: w,
	}
	kub.msg.MsgType = cmd
	kub.msg.ID = env.MessageDefaultID

	if !kub.initClient() {
		logger.Error().Println(pkgName, "failed initialising Kubernetes client")
	}

	return &kub
}

func (obj *kubernet) Name() string {
	return cmd
}

func (obj *kubernet) execute() {
	services := obj.monitorServices()
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

	if obj.klient == nil {
		return fmt.Errorf("could not connect to kubernetes cluster")
	}

	go func() {
		ticker := time.NewTicker(updatePeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				obj.execute()
			}
		}
	}()
	return nil
}
