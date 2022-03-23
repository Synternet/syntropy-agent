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

func (c *Client) CreateTest() (*TwampTest, error) {
	start := new(StartSessions)
	start.Two = 2 // TODO: rename to command and use contants
	err := sendMessage(c.GetConnection(), start)
	if err != nil {
		return nil, err
	}

	sack := new(StartAck)
	err = receiveMessage(c.GetConnection(), sack)
	if err != nil {
		return nil, err
	}

	err = checkAcceptStatus(sack.Accept, "test setup")
	if err != nil {
		return nil, err
	}

	test := &TwampTest{session: c}
	remoteAddr, err := test.RemoteAddr()
	if err != nil {
		return nil, err
	}

	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", test.GetLocalTestHost(), c.GetConfig().Port))
	if err != nil {
		return nil, err
	}

	// Create new connection for test
	conn, err := net.DialUDP("udp", localAddr, remoteAddr)
	if err != nil {
		return nil, err
	}
	err = test.SetConnection(conn)
	if err != nil {
		return nil, err
	}

	return test, nil
}

func (c *Client) stopSession() error {
	req := new(StopSessions)
	req.Three = 3 // TODO const
	req.Accept = AcceptOK
	req.Number = 1 // Stop single session
	return sendMessage(c.GetConnection(), req)
}
