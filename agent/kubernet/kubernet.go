package kubernet

import (
	"fmt"
	"io"

	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/slock"
	"golang.org/x/build/kubernetes"
)

const (
	pkgName = "Kubernetes. "
	cmd     = "KUBERNETES_SERVICE_INFO"
)

type kubernet struct {
	slock.AtomicServiceLock
	writer io.Writer
	kubcl  *kubernetes.Client
}

func New(w io.Writer) common.Service {
	kub := kubernet{
		writer: w,
	}

	return &kub
}

func (obj *kubernet) Name() string {
	return cmd
}

func (obj *kubernet) Start() error {
	if !obj.TryLock() {
		return fmt.Errorf("kubernetes watcher already running")
	}
	return nil
}

func (obj *kubernet) Stop() error {
	if !obj.TryUnlock() {
		return fmt.Errorf("kubernetes watcher is not running")
	}
	return nil
}
