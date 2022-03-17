package twamp

import (
	"encoding/binary"
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

func (s *Session) CreateTest() (*TwampTest, error) {
	var pdu []byte = make([]byte, 32)
	pdu[0] = 2

	s.Write(pdu)

	startAckBuffer, err := readFromSocket(s.GetConnection(), 32)
	if err != nil {
		log.Printf("Cannot read: %s\n", err)
		return nil, err
	}

	accept, err := startAckBuffer.ReadByte()
	if err != nil {
		log.Printf("Cannot read: %s\n", err)
		return nil, err
	}

	err = checkAcceptStatus(accept, "test setup")
	if err != nil {
		return nil, err
	}

	test := &TwampTest{session: s}
	remoteAddr, err := test.RemoteAddr()
	if err != nil {
		return nil, err
	}
	localAddress := fmt.Sprintf("%s:%d", test.GetLocalTestHost(), s.GetConfig().Port)
	localAddr, err := net.ResolveUDPAddr("udp", localAddress)
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

func (s *Session) Stop() {
	//	log.Println("Stopping test sessions.")
	var pdu []byte = make([]byte, 32)
	pdu[0] = byte(3)                       // Stop-Sessions Command Number
	pdu[1] = byte(0)                       // Accept Status (0 = OK)
	binary.BigEndian.PutUint16(pdu[4:], 1) // Number of Sessions
	s.GetConnection().Write(pdu)
}
