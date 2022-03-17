package twamp

import (
	"fmt"
	"log"
	"net"
)

type Session struct {
	conn   *Client
	port   uint16
	config SessionConfig
}

func (s *Session) GetConnection() net.Conn {
	return s.conn.GetConnection()
}

func (s *Session) GetConfig() SessionConfig {
	return s.config
}

func (s *Session) Write(buf []byte) {
	s.GetConnection().Write(buf)
}

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

func (s *Session) CreateTest() (*TwampTest, error) {
	start := new(StartSessions)
	start.Two = 2 // TODO: rename to command and use contants
	err := sendMessage(s.GetConnection(), start)
	if err != nil {
		return nil, err
	}

	sack := new(StartAck)
	err = receiveMessage(s.GetConnection(), sack)
	if err != nil {
		return nil, err
	}

	err = checkAcceptStatus(sack.Accept, "test setup")
	if err != nil {
		return nil, err
	}

	test := &TwampTest{session: s}
	remoteAddr, err := test.RemoteAddr()
	if err != nil {
		return nil, err
	}

	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", test.GetLocalTestHost(), s.GetConfig().Port))
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", localAddr, remoteAddr)
	test.SetConnection(conn)

	if err != nil {
		log.Printf("Some error %+v", err)
		return nil, err
	}

	return test, nil
}

func (s *Session) Stop() error {
	req := new(StopSessions)
	req.Three = 3 // TODO const
	req.Accept = AcceptOK
	req.Number = 1 // Stop single session
	return sendMessage(s.GetConnection(), req)
}
