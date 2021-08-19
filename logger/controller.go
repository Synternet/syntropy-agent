package logger

import (
	"encoding/json"
	"io"
)

type loggerMessage struct {
	ID        string `json:"id"`
	MsgType   string `json:"type"`
	Timestamp string `json:"executed_at,omitempty"`
	Data      struct {
		Level   string `json:"severity"`
		Message string `json:"message"`
	}
}

type controllerLogger struct {
	wr    io.Writer
	level string
}

func (l *controllerLogger) Write(b []byte) (n int, err error) {
	msg := loggerMessage{
		ID:      "-",
		MsgType: "LOGGER",
	}

	msg.Data.Message = string(b)

	raw, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}

	return l.wr.Write(raw)
}
