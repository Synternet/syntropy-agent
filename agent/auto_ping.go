package agent

import "encoding/json"

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

func autoPing(a *Agent, raw []byte) (resp []byte, err error) {

	var pingReq autoPingRequest
	err = json.Unmarshal(raw, &pingReq)

	// TODO: implement me
	return resp, err
}
