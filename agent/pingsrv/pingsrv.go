package pingsrv

import (
	"context"
	"fmt"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/twamp"
)

const (
	pkgName = "TwampServer. "
	cmd     = "TWAMP"
)

type pingServer struct {
	twampServer *twamp.Server
}

func New() common.Service {
	var err error
	obj := pingServer{}
	obj.twampServer, err = twamp.NewServer("0.0.0.0", 2000)
	if err != nil {
		logger.Error().Println(pkgName, "Creating TWAMP server:", err)
	}

	return &obj
}
func (obj *pingServer) Name() string {
	return cmd
}

func (obj *pingServer) Run(ctx context.Context) error {
	if obj.twampServer == nil {
		return fmt.Errorf("TWAMP server is not created")
	}

	go func() {
		err := obj.twampServer.Serve(ctx)
		//check if this is real error or context was closed
		select {
		case <-ctx.Done():
			return
		default:
			logger.Error().Println(pkgName, "TWAMP server terminated:", err)
		}
	}()

	return nil
}
