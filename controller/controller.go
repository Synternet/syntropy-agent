package controller

import (
	"io"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/env"
)

type Controller interface {
	// The primary idea was to use Reader interface here
	// But the reader may read a partial message and this will complicate agent main loop
	// and agent would be responsible for receiving and joining all message parts.
	// So instead hide that joining part and make a controller responsible for receiving full message.
	Recv() ([]byte, error)
	// Writer method Write(b) is used for sending message to controller
	io.Writer
	// Close() terminates controller. After Close controller will not reconnect
	// and may bot be used to  receive or send messages.
	io.Closer
}

// Command interface is used for controller commands executors
type Command interface {
	Name() string
	Exec(data []byte) error
}

// Service interface describes background running instances
type Service interface {
	Name() string
	Start() error
	Stop() error
}

// Generic message struct (common part for all messages)
type MessageHeader struct {
	ID        string `json:"id"`
	MsgType   string `json:"type"`
	Timestamp string `json:"executed_at,omitempty"`
}

func (mh *MessageHeader) Now() {
	mh.Timestamp = time.Now().Format(env.TimeFormat)
}

type ErrorResponce struct {
	MessageHeader
	Data struct {
		Type    string `json:"type"`
		Message string `json:"error"`
	} `json:"data"`
}
