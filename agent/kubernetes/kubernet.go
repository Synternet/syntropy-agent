package kubernetes

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/slock"
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
	slock.AtomicServiceLock
	writer io.Writer
	klient *kubernetes.Clientset
	msg    kubernetesInfoMessage
	ticker *time.Ticker
	stop   chan bool
}

func New(w io.Writer) common.Service {
	kub := kubernet{
		writer: w,
		stop:   make(chan bool),
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

func (obj *kubernet) Start() error {
	if !obj.TryLock() {
		return fmt.Errorf("kubernetes watcher already running")
	}

	if obj.klient == nil {
		return fmt.Errorf("could not connect to kubernetes cluster")
	}

	obj.ticker = time.NewTicker(updatePeriod)
	go func() {
		for {
			select {
			case <-obj.stop:
				return
			case <-obj.ticker.C:
				obj.execute()
			}
		}
	}()
	return nil
}

func (obj *kubernet) Stop() error {
	if !obj.TryUnlock() {
		return fmt.Errorf("kubernetes watcher is not running")
	}

	if obj.klient == nil {
		return fmt.Errorf("could not connect to kubernetes cluster")
	}

	obj.ticker.Stop()
	obj.stop <- true

	return nil
}