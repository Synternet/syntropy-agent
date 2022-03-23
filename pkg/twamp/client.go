package twamp

import (
	"net"
)

type SessionConfig struct {
	Port    int
	Padding int
	Timeout int
	TOS     int
}

type Client struct {
	host string
	port uint16
	conn net.Conn

	config SessionConfig
}

func NewClient(hostname string, config SessionConfig) (*Client, error) {
	// connect to remote host
	conn, err := net.Dial("tcp", hostname)
	if err != nil {
		return nil, err
	}

	// create a new Connection
	client := &Client{
		host:   hostname,
		conn:   conn,
		config: config,
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

	err = client.createSession()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) GetConnection() net.Conn {
	return c.conn
}

func (c *Client) Close() error {
	c.stopSession()
	return c.GetConnection().Close()
}

func (c *Client) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Client) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// TODO do we need these ?
func (c *Client) GetConfig() SessionConfig {
	return c.config
}

func (c *Client) Write(buf []byte) {
	c.GetConnection().Write(buf)
}
