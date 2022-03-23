package twamp

import (
	"fmt"
	"net"
)

type StartSessions struct {
	Two  byte
	MBZ  [15]byte
	HMAC [16]byte
}

type StartAck struct {
	Accept byte
	MBZ    [15]byte
	HMAC   [16]byte
}

type StopSessions struct {
	Three  byte
	Accept byte
	MBZ    [2]byte
	Number uint32
	MBZ2   [8]byte
}

func (c *Client) createTest() error {
	start := new(StartSessions)
	start.Two = 2 // TODO: rename to command and use contants
	err := sendMessage(c.conn, start)
	if err != nil {
		return err
	}

	sack := new(StartAck)
	err = receiveMessage(c.conn, sack)
	if err != nil {
		return err
	}

	err = checkAcceptStatus(sack.Accept, "test setup")
	if err != nil {
		return err
	}

	c.test = &twampTest{session: c}
	remoteAddr, err := c.remoteTestAddr()
	if err != nil {
		return err
	}

	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", c.localTestHost(), c.config.LocalPort))
	if err != nil {
		return err
	}

	// Create new connection for test
	conn, err := net.DialUDP("udp", localAddr, remoteAddr)
	if err != nil {
		return err
	}
	err = c.test.setConnection(conn)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) stopSession() error {
	req := new(StopSessions)
	req.Three = 3 // TODO const
	req.Accept = AcceptOK
	req.Number = 1 // Stop single session
	return sendMessage(c.conn, req)
}
