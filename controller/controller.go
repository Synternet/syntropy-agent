package controller

import (
	"io"
)

type Controller interface {
	// Open function does additional initialisation to controller (for those that need it)
	// E.g. its nice to have late connecting to SaaS controller,
	// because you cannot do long delays in WebSocket connection,
	// because it results in WebSockets timeout
	Open() error
	// The primary idea was to use Reader interface here
	// But the reader may read a partial message and this will complicate agent main loop
	// and agent would be responsible for receiving and joining all message parts.
	// So instead hide that joining part and make a controller responsible for receiving full message.
	Recv() ([]byte, error)
	// Writer method Write(b) is used for sending message to controller
	io.Writer
	// Close() terminates controller. After Close controller will not reconnect
	// and may not be used to receive or send messages.
	io.Closer
}
