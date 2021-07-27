package agent

import (
	"encoding/json"
	"fmt"
	"log"
)

func (a *Agent) processCommand(raw []byte) error {
	var req messageHeader
	if err := json.Unmarshal(raw, &req); err != nil {
		return fmt.Errorf("json unmarshal error: %s", err.Error())
	}

	functionCall, ok := a.commands[req.MsgType]
	if !ok {
		return fmt.Errorf("command '%s' not found", req.MsgType)
	}

	// TODO process and send back responce
	log.Println("Calling ", req.MsgType, req.ID)
	log.Println(string(raw))

	if err := functionCall(a, raw); err != nil {
		return fmt.Errorf("error while executing `%s` commant: %s",
			req.ID, err.Error())
	}
	log.Println(req.MsgType, "completed")

	return nil
}
