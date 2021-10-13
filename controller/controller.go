package controller

import (
	"context"
	"fmt"

	"github.com/SyntropyNet/syntropy-agent-go/controller/blockchain"
	"github.com/SyntropyNet/syntropy-agent-go/controller/saas"
	"github.com/SyntropyNet/syntropy-agent-go/controller/script"
	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
)

func New(ctx context.Context, contype int) (controller common.Controller, err error) {
	switch contype {
	case config.ControllerSaas:
		controller, err = saas.New(ctx)
	case config.ControllerScript:
		controller, err = script.New(ctx)
	case config.ControllerBlockchain:
		controller, err = blockchain.New(ctx)
	default:
		err = fmt.Errorf("unexpected controller type %d", contype)
	}
	return
}
