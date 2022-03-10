package twamp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Connect(hostname string) (*Connection, error) {
	// connect to remote host
	conn, err := net.Dial("tcp", hostname)
	if err != nil {
		return nil, err
	}

	// create a new Connection
	Connection := NewConnection(conn)

	// check for greeting message from TWAMP server
	greeting, err := Connection.getTwampServerGreetingMessage()
	if err != nil {
		return nil, err
	}

	// check greeting mode for errors
	switch greeting.Mode {
	case ModeUnspecified:
		return nil, errors.New("The TWAMP server is not interested in communicating with you.")
	case ModeUnauthenticated:
	case ModeAuthenticated:
		return nil, errors.New("Authentication is not currently supported.")
	case ModeEncypted:
		return nil, errors.New("Encyption is not currently supported.")
	}

	// negotiate TWAMP session configuration
	Connection.sendTwampClientSetupResponse()

	// check the start message from TWAMP server
	serverStartMessage, err := Connection.getTwampServerStartMessage()
	if err != nil {
		return nil, err
	}

	err = checkAcceptStatus(int(serverStartMessage.Accept), "connection")
	if err != nil {
		return nil, err
	}

	return Connection, nil
}

func readFromSocket(reader io.Reader, size int) (bytes.Buffer, error) {
	buf := make([]byte, size)
	buffer := *bytes.NewBuffer(buf)
	bytesRead, err := reader.Read(buf)

	if err != nil && bytesRead < size {
		return buffer, errors.New(fmt.Sprintf("readFromSocket: expected %d bytes, got %d", size, bytesRead))
	}

	return buffer, err
}

/*
	Convenience function for checking the accept code contained in various TWAMP server
	response messages.
*/
func checkAcceptStatus(accept int, cmd string) error {
	switch accept {
	case AcceptOK:
		return nil
	case AcceptFailure:
		return errors.New(fmt.Sprintf("ERROR: The ", cmd, " failed."))
	case AcceptInternalError:
		return errors.New(fmt.Sprintf("ERROR: The ", cmd, " failed: internal error."))
	case AcceptNotSupported:
		return errors.New(fmt.Sprintf("ERROR: The ", cmd, " failed: not supported."))
	case AcceptPermResLimitation:
		return errors.New(fmt.Sprintf("ERROR: The ", cmd, " failed: permanent resource limitation."))
	case AcceptTempResLimitation:
		return errors.New(fmt.Sprintf("ERROR: The ", cmd, " failed: temporary resource limitation."))
	}
	return nil
}
