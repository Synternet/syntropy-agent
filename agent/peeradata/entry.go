package peeradata

import (
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
)

type Entry struct {
	PreviousConnID int    `json:"prev_connection_id,omitempty"`
	ConnectionID   int    `json:"connection_id"`
	GroupID        int    `json:"connection_group_id,omitempty"`
	Timestamp      string `json:"timestamp"`
}

func NewEntry(prevID, connID, grID int) *Entry {
	return &Entry{
		PreviousConnID: prevID,
		ConnectionID:   connID,
		GroupID:        grID,
		Timestamp:      time.Now().Format(env.TimeFormat),
	}
}
