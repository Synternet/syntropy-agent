package hostnetsrv

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/scontext"
)

const (
	pkgName      = "HostNetServices. "
	cmd          = "HW_SERVICE_INFO"
	updatePeriod = time.Second * 5
)

type hostNetServices struct {
	writer io.Writer
	msg    hostNetworkServicesMessage
	ctx    scontext.StartStopContext
}

func New(ctx context.Context, w io.Writer) common.Service {
	obj := hostNetServices{
		writer: w,
		ctx:    scontext.New(ctx),
	}
	obj.msg.MsgType = cmd
	obj.msg.ID = env.MessageDefaultID
	return &obj
}

func (obj *hostNetServices) Name() string {
	return cmd
}

func (obj *hostNetServices) Start() error {
	ctx, err := obj.ctx.CreateContext()
	if err != nil {
		return fmt.Errorf("host network services watcher already running")
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

func (obj *hostNetServices) Stop() error {
	if err := obj.ctx.CancelContext(); err != nil {
		return fmt.Errorf("host network services watcher is not running")
	}

	return nil
}
