package twamp

import (
	"bytes"
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
	err = recvServerGreeting(Connection.GetConnection())
	if err != nil {
		return nil, err
	}

	// negotiate TWAMP session configuration
	err = sendClientSetupResponse(Connection.GetConnection())
	if err != nil {
		return nil, err
	}

	// check the start message from TWAMP server
	serverStartMessage, err := Connection.getServerStartMessage()
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
		return buffer, fmt.Errorf("readFromSocket: expected %d bytes, got %d", size, bytesRead)
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
		return fmt.Errorf("ERROR: The %s failed", cmd)
	case AcceptInternalError:
		return fmt.Errorf("ERROR: The %s failed: internal error", cmd)
	case AcceptNotSupported:
		return fmt.Errorf("ERROR: The %s failed: not supported", cmd)
	case AcceptPermResLimitation:
		return fmt.Errorf("ERROR: The %s failed: permanent resource limitation", cmd)
	case AcceptTempResLimitation:
		return fmt.Errorf("ERROR: The %s failed: temporary resource limitation", cmd)
	}
	return nil
}
