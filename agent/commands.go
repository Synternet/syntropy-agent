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
	_, err := functionCall(a, raw)
	if err != nil {
		return fmt.Errorf("error while executing `%s` commant: %s",
			req.ID, err.Error())
	}

	return nil
}

func autoPing(a *Agent, raw []byte) (resp []byte, err error) {

	var pingReq autoPingRequest
	err = json.Unmarshal(raw, &pingReq)

	log.Println("Calling autoPing", pingReq)
	return resp, err
}

func getInfo(a *Agent, raw []byte) (resp []byte, err error) {

	var req getInfoRequest
	err = json.Unmarshal(raw, &req)

	log.Println("Calling getInfo", req)
	return resp, err
}
