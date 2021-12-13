package settings

import (
	"encoding/json"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
)

type setSettings struct {
}

func New() common.Command {
	return &setSettings{}
}

func (s *setSettings) Name() string {
	return cmd
}

func (s *setSettings) Exec(raw []byte) error {
	var req settingsMessage
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	config.SetRerouteThresholds(req.Data.Thresholds.Diff, req.Data.Thresholds.Ratio)
	return nil
}
