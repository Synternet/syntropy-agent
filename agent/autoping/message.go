package autoping

import (
	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
)

type autoPingRequest struct {
	common.MessageHeader
	Data struct {
		IPs       []string `json:"ips"`
		Interval  int      `json:"interval"`
		RespLimit int      `json:"response_limit"`
	} `json:"data"`
}

type pingResponseEntry struct {
	IP      string  `json:"ip"`
	Latency float32 `json:"latency_ms,omitempty"`
	Loss    float32 `json:"packet_loss"`
}

type autoPingResponse struct {
	common.MessageHeader
	Data struct {
		Pings []pingResponseEntry `json:"pings"`
	} `json:"data"`
}

func newResponceMsg() autoPingResponse {
	msg := autoPingResponse{}
	msg.Data.Pings = []pingResponseEntry{}
	msg.MsgType = cmd
	msg.ID = env.MessageDefaultID
	msg.Now()

	return msg
}
