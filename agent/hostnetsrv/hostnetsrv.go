package hostnetsrv

import (
	"context"
	"io"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
)

const (
	pkgName = "HostNetServices. "
	cmd     = "HW_SERVICE_INFO"
)

type hostNetServices struct {
	writer io.Writer
	msg    hostNetworkServicesMessage
}

func New(w io.Writer) common.Service {
	obj := hostNetServices{
		writer: w,
	}
	obj.msg.MsgType = cmd
	obj.msg.ID = env.MessageDefaultID
	return &obj
}

func (obj *hostNetServices) Name() string {
	return cmd
}

func (obj *hostNetServices) Run(ctx context.Context) error {
	go func() {
		ticker := time.NewTicker(config.PeerCheckTime())
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
