package agent

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/SyntropyNet/syntropy-agent-go/config"
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

	_, err := functionCall(a, raw)
	if err != nil {
		return fmt.Errorf("error while executing `%s` commant: %s",
			req.ID, err.Error())
	}
	log.Println(req.MsgType, "completed")

	return nil
}

func autoPing(a *Agent, raw []byte) (resp []byte, err error) {

	var pingReq autoPingRequest
	err = json.Unmarshal(raw, &pingReq)

	return resp, err
}

func getInfo(a *Agent, raw []byte) (rv []byte, err error) {

	var req getInfoRequest
	err = json.Unmarshal(raw, &req)
	if err != nil {
		return
	}

	resp := getInfoResponce{
		messageHeader: req.messageHeader,
	}
	resp.Data.Provider = config.GetAgentProvider()
	resp.Data.Status = config.GetServicesStatus()
	resp.Data.Tags = config.GetAgentTags()
	resp.Data.ExternalIP = config.GetPublicIp()
	resp.Data.NetworkInfo = FakeNetworkInfo()
	resp.Data.ContainerInfo = FakeContainerInfo()

	arr, err := json.Marshal(&resp)
	if err != nil {
		log.Println("Marshal error: ", err)
		return
	}

	a.Transmit(arr)

	return rv, err
}
