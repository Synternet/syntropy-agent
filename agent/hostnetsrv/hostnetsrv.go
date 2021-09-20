package hostnetsrv

import (
	"fmt"
	"io"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/slock"
)

const (
	pkgName      = "HostNetServices. "
	cmd          = "HW_SERVICE_INFO"
	updatePeriod = time.Second * 5
)

type hostNetServices struct {
	slock.AtomicServiceLock
	writer io.Writer
	msg    hostNetworkServicesMessage
	ticker *time.Ticker
	stop   chan bool
}

func New(w io.Writer) common.Service {
	obj := hostNetServices{
		writer: w,
		stop:   make(chan bool),
	}
	obj.msg.MsgType = cmd
	obj.msg.ID = env.MessageDefaultID
	return &obj
}

func (obj *hostNetServices) Name() string {
	return cmd
}

func (obj *hostNetServices) Start() error {
	if !obj.TryLock() {
		return fmt.Errorf("host network services watcher already running")
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

func (obj *hostNetServices) Stop() error {
	if !obj.TryUnlock() {
		return fmt.Errorf("host network services watcher is not running")
	}

	obj.ticker.Stop()
	obj.stop <- true

	return nil
}
