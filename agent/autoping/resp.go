package autoping

import (
	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
)

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

func (resp *autoPingResponse) PingProcess(data *multiping.PingData) {
	// TODO: respect controllers set LimitCount
	data.Iterate(func(ip string, val multiping.PingStats) {
		resp.Data.Pings = append(resp.Data.Pings,
			pingResponseEntry{
				IP:      ip,
				Latency: val.Latency(),
				Loss:    val.Loss(),
			})
	})

	// Clear old statistics
	data.Reset()
}
