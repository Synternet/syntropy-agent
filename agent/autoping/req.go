package autoping

import "github.com/SyntropyNet/syntropy-agent/agent/common"

type autoPingRequest struct {
	common.MessageHeader
	Data struct {
		IPs       []string `json:"ips"`
		Interval  int      `json:"interval"`
		RespLimit int      `json:"response_limit"`
	} `json:"data"`
}
