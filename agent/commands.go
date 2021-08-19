package agent

import (
	"encoding/json"

	"github.com/SyntropyNet/syntropy-agent-go/logger"
)

func (a *Agent) processCommand(raw []byte) {
	var req messageHeader
	if err := json.Unmarshal(raw, &req); err != nil {
		logger.Error().Println(pkgName, "json message unmarshal error: ", err)
		return
	}

	functionCall, ok := a.commands[req.MsgType]
	if !ok {
		logger.Error().Printf("%s Command '%s' not found\n", pkgName, req.MsgType)
		return
	}

	logger.Debug().Println(pkgName, "Received: ", string(raw))

	err := functionCall(a, raw)
	if err != nil {
		logger.Error().Printf("%s Command '%s' failed: %s\n", pkgName, req.ID, err.Error())
	}
	logger.Info().Printf("%s Command '%s' completed.", pkgName, req.ID)
}
