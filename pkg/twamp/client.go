package twamp

import (
	"fmt"
	"net"
)

type Client struct {
	host     string
	testPort uint16

	controlConn net.Conn
	//	testConn    *net.UDPConn

	test   *twampTest
	config *clientConfig
}

func NewClient(hostname string, opts ...clientOption) (*Client, error) {
	// connect to remote host
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", hostname, TwampControlPort))
	if err != nil {
		return nil, err
	}

	// create a new Connection
	client := &Client{
		host:        hostname,
		controlConn: conn,
		config: &clientConfig{
			LocalPort:     0,
			PaddingSize:   0,
			PaddingZeroes: false,
			Timeout:       1,
			TOS:           0,
		},
	}

	for _, opt := range opts {
		opt(client.config)
	}

	// check for greeting message from TWAMP server
	err = recvServerGreeting(client.controlConn)
	if err != nil {
		return nil, err
	}

	// negotiate TWAMP session configuration
	err = sendClientSetupResponse(client.controlConn)
	if err != nil {
		return nil, err
	}

	// check the start message from TWAMP server
	err = recvServerStartMessage(client.controlConn)
	if err != nil {
		return nil, err
	}

	err = client.createSession()
	if err != nil {
		return nil, err
	}

	err = client.createTest()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) PaddingSize() uint {
	return uint(c.config.PaddingSize)
}

func (c *Client) GetHost() string {
	return c.host
}

func (c *Client) GetStats() *Statistics {
	return &c.test.stats
}

func (c *Client) Close() error {
	c.stopSession()
	return c.controlConn.Close()
}

func (c *Client) LocalAddr() net.Addr {
	return c.controlConn.LocalAddr()
}

func (c *Client) RemoteAddr() net.Addr {
	return c.controlConn.RemoteAddr()
}
