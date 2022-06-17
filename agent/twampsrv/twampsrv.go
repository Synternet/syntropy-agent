package twampsrv

import (
	"context"
	"fmt"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/state"
	"github.com/SyntropyNet/syntropy-agent/pkg/twamp"
)

const (
	pkgName = "TwampServer. "
	cmd     = "TWAMP"
)
const (
	// State machine constants
	stopped = iota
	running
)

type twampServer struct {
	state.StateMachine
	server *twamp.Server
}

func New() common.Service {
	var err error
	obj := twampServer{}
	obj.server, err = twamp.NewServer("0.0.0.0", 2000)
	if err != nil {
		logger.Error().Println(pkgName, "Creating TWAMP server:", err)
	}
	obj.SetState(stopped)

	return &obj
}
func (obj *twampServer) Name() string {
	return cmd
}

func (obj *twampServer) Run(ctx context.Context) error {
	if obj.server == nil {
		return fmt.Errorf("TWAMP server is not created")
	}
	obj.SetState(running)

	go func() {
		<-ctx.Done()
		logger.Info().Println(pkgName, "stopping", cmd)
		obj.SetState(stopped)
		obj.server.Close()
	}()

	go func() {
		err := obj.server.Run()
		//check if this is real error or context was closed
		if obj.GetState() == running {
			logger.Error().Println(pkgName, "TWAMP server terminated:", err)
		}
	}()

	return nil
}
