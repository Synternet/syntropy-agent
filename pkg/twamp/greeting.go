package twamp

import (
	"crypto/rand"
	"fmt"
	"net"
)

type ServerGreeting struct {
	Unused    [12]byte
	Modes     uint32
	Challenge [16]byte
	Salt      [16]byte
	Count     uint32
	MBZ       [12]byte
}

func sendServerGreeting(conn net.Conn) error {
	greeting, err := createServerGreeting(ModeUnauthenticated)
	if err != nil {
		return err
	}

	return sendMessage(conn, greeting)
}

func createServerGreeting(modes uint32) (*ServerGreeting, error) {
	greeting := new(ServerGreeting)

	greeting.Modes = modes
	greeting.Count = 1024

	_, err := rand.Read(greeting.Challenge[:])
	if err != nil {
		return nil, err
	}

	_, err = rand.Read(greeting.Salt[:])
	if err != nil {
		return nil, err
	}

	return greeting, nil
}

func (c *Client) recvServerGreeting() error {
	// check for greeting message from TWAMP server
	greeting := new(ServerGreeting)
	err := receiveMessage(c.controlConn, greeting)
	if err != nil {
		return err
	}

	// check greeting mode for errors
	switch greeting.Modes {
	case ModeUnauthenticated:
		// The only mode currently supported
		return nil
	case ModeUnspecified:
		return fmt.Errorf("TWAMP server is not interested in communicating with you")
	case ModeAuthenticated:
		return fmt.Errorf("authentication is not currently supported")
	case ModeEncypted:
		return fmt.Errorf("encyption is not currently supported")
	default:
		return fmt.Errorf("unsupported mode 0x%x", greeting.Modes)
	}
}

// TWAMP client session negotiation message.
type SetUpResponse struct {
	Mode     uint32
	KeyID    [80]byte
	Token    [64]byte
	ClientIV [16]byte
}

func (c *Client) sendClientSetupResponse() error {
	// negotiate TWAMP session configuration
	response := &SetUpResponse{
		Mode: ModeUnauthenticated,
	}
	return sendMessage(c.controlConn, response)
}

type ServerStart struct {
	MBZ       [15]byte
	Accept    byte
	ServerIV  [16]byte
	StartTime Timestamp
	MBZ2      [8]byte
}

func (c *Client) recvServerStartMessage() error {
	srvstart := new(ServerStart)

	err := receiveMessage(c.controlConn, srvstart)
	if err != nil {
		return err
	}

	err = checkAcceptStatus(srvstart.Accept, "connection")
	if err != nil {
		return err
	}

	return nil
}
