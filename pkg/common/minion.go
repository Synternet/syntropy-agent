package common

import "github.com/SyntropyNet/syntropy-agent-go/pkg/slock"

// Command interface is used for controller commands executors
type Command interface {
	Name() string
	Exec(data []byte) error
}

// Service interface describes background running instances
type Service interface {
	slock.ServiceLocker
	Name() string
	Start() error
	Stop() error
}

// CommandService implements both: Command + Service
// Is intendend for those controller commands, that must have background task running
type CommandService interface {
	Command
	Service
}
