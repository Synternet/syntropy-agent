package agent

import (
	"encoding/json"
	"log"
)

func (a *Agent) processCommand(raw []byte) {
	var req messageHeader
	if err := json.Unmarshal(raw, &req); err != nil {
		log.Println("json unmarshal error: ", err)
		return
	}

	functionCall, ok := a.commands[req.MsgType]
	if !ok {
		log.Printf("command '%s' not found\n", req.MsgType)
		return
	}

	// TODO process and send back responce
	log.Println("Calling ", req.MsgType, req.ID)
	log.Println(string(raw))

	err := functionCall(a, raw)
	if err != nil {
		log.Printf("error while executing `%s` command: %s\n",
			req.ID, err.Error())
	}
}
