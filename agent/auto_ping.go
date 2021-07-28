package agent

import (
	"encoding/json"
	"time"
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
		Pings []struct {
			IP      string  `json:"ip"`
			Latency int     `json:"latency_ms"`
			Loss    float32 `json:"packet_loss"`
		} `json:"pings"`
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
