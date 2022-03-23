package twamp

import (
	"fmt"
	"net"
)

type Client struct {
	host     string
	testPort uint16

	controlConn net.Conn
	testConn    *net.UDPConn

	testSequence uint32
	stats        Statistics
	config       *clientConfig
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
	err = client.recvServerGreeting()
	if err != nil {
		return nil, err
	}

	// negotiate TWAMP session configuration
	err = client.sendClientSetupResponse()
	if err != nil {
		return nil, err
	}

	// check the start message from TWAMP server
	err = client.recvServerStartMessage()
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

func (c *Client) GetHost() string {
	return c.host
}

func (c *Client) Stats() *Statistics {
	return &c.stats
}

func (c *Client) Close() error {
	c.stopSession()
	return c.controlConn.Close()
}
