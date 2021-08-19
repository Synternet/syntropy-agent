package agent

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pinger"
)

type autoPingRequest struct {
	messageHeader
	Data struct {
		IPs       []string `json:"ips"`
		Interval  int      `json:"interval"`
		RespLimit int      `json:"responce_limit"`
	} `json:"data"`
}

type autoPingResponce struct {
	messageHeader
	Data struct {
		Pings []pinger.PingResult `json:"pings"`
	} `json:"data"`
}

func autoPing(a *Agent, raw []byte) error {

	var req autoPingRequest
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	a.ping.Stop()
	a.ping.Setup(time.Duration(req.Data.Interval)*time.Second, req.Data.RespLimit)
	a.ping.AddHost(req.Data.IPs...)
	a.ping.Start()

	return nil
}

func (a *Agent) ProcessPingResults(pr []pinger.PingResult) {
	var resp autoPingResponce
	resp.Data.Pings = pr
	resp.MsgType = "AUTO_PING"
	resp.ID = "ID." + strconv.FormatInt(time.Now().Unix(), 10)
	resp.Now()

	if len(resp.Data.Pings) > 0 {
		arr, err := json.Marshal(resp)
		if err != nil {
			logger.Error().Println(pkgName, "Process Ping Results: ", err)
			return
		}

		a.Write(arr)
	}
}
