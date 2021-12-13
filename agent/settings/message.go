package settings

import (
	"github.com/SyntropyNet/syntropy-agent/agent/common"
)

const cmd = "SET_SETTINGS"
const pkgName = "SetSettings. "

type thresholdsEntry struct {
	Diff  float32 `json:"latency_diff"`
	Ratio float32 `json:"latency_ratio"`
}

type settingsMessage struct {
	common.MessageHeader
	Data struct {
		Thresholds thresholdsEntry `json:"rerouting_threshold"`
	} `json:"data"`
}
