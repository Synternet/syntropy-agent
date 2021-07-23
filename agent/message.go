package agent

import "time"

// Generic message struct (common part for all messages)
type messageHeader struct {
	ID        string `json:"id"`
	MsgType   string `json:"type"`
	Timestamp string `json:"executed_at,omitempty"`
}

func (mh *messageHeader) Now() {
	mh.Timestamp = time.Now().Format("2006-01-02T15:04:05 -07:00")
}

type errorResponce struct {
	messageHeader
	Data struct {
		Type    string `json:"type"`
		Message string `json:"error"`
	} `json:"data"`
}
