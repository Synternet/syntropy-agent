package agent

import (
	"encoding/json"

	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
)

func (a *Agent) addCommand(cmd common.Command) error {
	a.commands[cmd.Name()] = cmd
	return nil
}

func (a *Agent) processCommand(raw []byte) {
	var req common.MessageHeader
	if err := json.Unmarshal(raw, &req); err != nil {
		logger.Error().Println(pkgName, "json message unmarshal error: ", err)
		return
	}

	cmd, ok := a.commands[req.MsgType]
	if !ok {
		logger.Error().Printf("%s Command '%s' not found\n", pkgName, req.MsgType)
		return
	}

	err := cmd.Exec(raw)
	if err != nil {
		logger.Error().Printf("%s Command '%s' failed: %s\n", pkgName, req.MsgType, err.Error())
	}
	logger.Info().Printf("%s Command '%s' completed.", pkgName, req.MsgType)
}