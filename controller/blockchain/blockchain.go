package blockchain

import (
	"errors"

	"github.com/SyntropyNet/syntropy-agent/controller"
)

// Blockchain controller. To be implemented in future
type BlockchainControler struct {
}

var err = errors.New("blockchain controller not yet implemented")

func New() (controller.Controller, error) {
	return nil, err
}

func (bcc *BlockchainControler) Recv() ([]byte, error) {
	return nil, err
}

func (bcc *BlockchainControler) Write(b []byte) (n int, err error) {
	return 0, err
}

func (bcc *BlockchainControler) Close() error {
	return err
}
