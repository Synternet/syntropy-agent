package agent

// Generic message struct (common part for all messages)
type messageHeader struct {
	ID      string `json:"id"`
	MsgType string `json:"type"`
}

type errorResponce struct {
	messageHeader
	Data struct {
		Type    string `json:"type"`
		Message string `json:"error"`
	} `json:"data"`
}
