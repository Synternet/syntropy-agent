package autoping

import "github.com/SyntropyNet/syntropy-agent/agent/common"

type AutoPingRequest struct {
	common.MessageHeader
	IPs       []string `json:"ips"`
	Interval  int      `json:"interval"`
	RespLimit int      `json:"response_limit"`
}
