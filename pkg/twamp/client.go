package twamp

import (
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

func (c *Client) GetConnection() net.Conn {
	return c.conn
}

func (c *Client) Close() {
	c.GetConnection().Close()
}

func (c *Client) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *Client) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
