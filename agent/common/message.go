package common

import (
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
)

// Generic message struct (common part for all messages)
type MessageHeader struct {
	ID        string `json:"id"`
	MsgType   string `json:"type"`
	Timestamp string `json:"executed_at,omitempty"`
}

func (mh *MessageHeader) Now() {
	mh.Timestamp = time.Now().Format(env.TimeFormat)
}

type ErrorResponse struct {
	MessageHeader
	Data struct {
		Type    string `json:"type"`
		Message string `json:"error"`
	} `json:"data"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
