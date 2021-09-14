package hostnetsrv

import (
	"fmt"
	"io"

	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/slock"
)

const (
	pkgName = "HostNetServices. "
	cmd     = "HW_SERVICE_INFO"
)

type hostNetServices struct {
	slock.AtomicServiceLock
	writer io.Writer
}

func New(w io.Writer) common.Service {
	obj := hostNetServices{
		writer: w,
	}

	return &obj
}

func (obj *hostNetServices) Name() string {
	return cmd
}

func (obj *hostNetServices) Start() error {
	if !obj.TryLock() {
		return fmt.Errorf("host network services watcher already running")
	}
	return nil
}

func (obj *hostNetServices) Stop() error {
	if !obj.TryUnlock() {
		return fmt.Errorf("host network services watcher is not running")
	}
	return nil
}
