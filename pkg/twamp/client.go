package twamp

import (
	"bytes"
	"fmt"
	"io"
	"net"
)

type Client struct {
	host string
	conn net.Conn
}

func NewClient(hostname string) (*Client, error) {
	// connect to remote host
	conn, err := net.Dial("tcp", hostname)
	if err != nil {
		return nil, err
	}

	// create a new Connection
	client := &Client{
		host: hostname,
		conn: conn,
	}

	// check for greeting message from TWAMP server
	err = recvServerGreeting(client.GetConnection())
	if err != nil {
		return nil, err
	}

	// negotiate TWAMP session configuration
	err = sendClientSetupResponse(client.GetConnection())
	if err != nil {
		return nil, err
	}

	// check the start message from TWAMP server
	err = recvServerStartMessage(client.GetConnection())
	if err != nil {
		return nil, err
	}

	return client, nil
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
