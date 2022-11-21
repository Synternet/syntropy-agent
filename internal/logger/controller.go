package logger

import (
	"encoding/json"
	"io"
	"time"

	"github.com/SyntropyNet/syntropy-agent/internal/env"
)

const cmd = "LOGGER"

type loggerMessage struct {
	ID        string `json:"id"`
	MsgType   string `json:"type"`
	Timestamp string `json:"executed_at,omitempty"`
	Data      struct {
		Level   string `json:"severity"`
		Message string `json:"message"`
	} `json:"data"`
}

type controllerLogger struct {
	wr    io.Writer
	level string
}

func (l *controllerLogger) Write(b []byte) (n int, err error) {
	msg := loggerMessage{
		ID:        env.MessageDefaultID,
		MsgType:   cmd,
		Timestamp: time.Now().Format(env.TimeFormat),
	}

	msg.Data.Message = string(b)
	msg.Data.Level = l.level
	raw, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}

	return l.wr.Write(raw)
}
