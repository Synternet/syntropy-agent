package twamp

import (
	"fmt"
	"net"
)

type Client struct {
	host     string
	testPort uint16
	conn     net.Conn

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
		host: hostname,
		conn: conn,
		config: &clientConfig{
			LocalPort: 0,
			Padding:   0,
			Timeout:   1,
			TOS:       0,
		},
	}

	for _, opt := range opts {
		opt(client.config)
	}

	// check for greeting message from TWAMP server
	err = recvServerGreeting(client.conn)
	if err != nil {
		return nil, err
	}

	// negotiate TWAMP session configuration
	err = sendClientSetupResponse(client.conn)
	if err != nil {
		return nil, err
	}

	// check the start message from TWAMP server
	err = recvServerStartMessage(client.conn)
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
	return uint(c.config.Padding)
}

func (c *Client) GetHost() string {
	return c.host
}

func (c *Client) GetStats() *Statistics {
	return &c.test.stats
}

func (c *Client) Close() error {
	c.stopSession()
	return c.conn.Close()
}

func (c *Client) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Client) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
