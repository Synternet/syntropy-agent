package twamp

import (
	"fmt"
	"net"

	"golang.org/x/net/ipv4"
)

type StartSessions struct {
	Command byte
	MBZ     [15]byte
	HMAC    [16]byte
}

type StartAck struct {
	Accept byte
	MBZ    [15]byte
	HMAC   [16]byte
}

type StopSessions struct {
	Command byte
	Accept  byte
	MBZ     [2]byte
	Number  uint32
	MBZ2    [8]byte
}

func (c *Client) createTest() error {
	start := new(StartSessions)
	start.Command = CmdStartTestSession
	err := sendMessage(c.controlConn, start)
	if err != nil {
		return err
	}

	sack := new(StartAck)
	err = receiveMessage(c.controlConn, sack)
	if err != nil {
		return err
	}

	err = checkAcceptStatus(sack.Accept, "test setup")
	if err != nil {
		return err
	}

	c.test = &twampTest{}
	remoteAddr, err := c.remoteTestAddr()
	if err != nil {
		return err
	}

	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", c.localTestHost(), c.config.LocalPort))
	if err != nil {
		return err
	}

	// Create new connection for test
	c.test.conn, err = net.DialUDP("udp", localAddr, remoteAddr)
	if err != nil {
		return err
	}

	// Configure test connection
	ipConn := ipv4.NewConn(c.test.conn)
	err = ipConn.SetTOS(c.config.TOS)
	if err != nil {
		return err
	}

	// RFC recommends IP TTL of 255
	err = ipConn.SetTTL(255)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) stopSession() error {
	req := new(StopSessions)
	req.Command = CmdStopSessions
	req.Accept = AcceptOK
	req.Number = 1 // Stop single session
	return sendMessage(c.controlConn, req)
}
