package peeradata

import (
	"time"

	"github.com/SyntropyNet/syntropy-agent/internal/env"
)

type Entry struct {
	PreviousConnID int    `json:"prev_connection_id"`
	ConnectionID   int    `json:"connection_id"`
	GroupID        int    `json:"connection_group_id"`
	Timestamp      string `json:"timestamp"`
	Reason         string `json:"route_change_reason,omitempty"`
}

func NewEntry(prevID, connID, grID int) *Entry {
	return &Entry{
		PreviousConnID: prevID,
		ConnectionID:   connID,
		GroupID:        grID,
		Timestamp:      time.Now().Format(env.TimeFormat),
	}
}
