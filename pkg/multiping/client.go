package multiping

import "github.com/SyntropyNet/syntropy-agent/pkg/multiping/pingdata"

// Unified interface to process ping data
type PingClient interface {
	PingProcess(pr *pingdata.PingData)
}
