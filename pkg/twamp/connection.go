package twamp

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
	"time"
)

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

type ServerStart struct {
	MBZ       [15]byte
	Accept    byte
	ServerIV  [16]byte
	StartTime Timestamp
	MBZ2      [8]byte
}

/* Byte offsets for Request-TW-Session TWAMP PDU */
const (
	command         = 0
	senderPort      = 12
	receiverPort    = 14
	paddingLength   = 64
	startTime       = 68
	timeout         = 76
	typePDescriptor = 84
)

type RequestTwSession []byte

func (b RequestTwSession) Encode(c SessionConfig) {
	start_time := NewTimestamp(time.Now())
	b[command] = byte(5)
	binary.BigEndian.PutUint16(b[senderPort:], 6666)
	binary.BigEndian.PutUint16(b[receiverPort:], uint16(c.Port))
	binary.BigEndian.PutUint32(b[paddingLength:], uint32(c.Padding))
	binary.BigEndian.PutUint32(b[startTime:], start_time.Seconds)
	binary.BigEndian.PutUint32(b[startTime+4:], start_time.Fraction)
	binary.BigEndian.PutUint32(b[timeout:], uint32(c.Timeout))
	binary.BigEndian.PutUint32(b[timeout+4:], 0)
	binary.BigEndian.PutUint32(b[typePDescriptor:], uint32(c.TOS))
}

func (c *Client) CreateSession(config SessionConfig) (*Session, error) {
	var pdu RequestTwSession = make(RequestTwSession, 112)

	pdu.Encode(config)

	c.GetConnection().Write(pdu)

	acceptBuffer, err := readFromSocket(c.GetConnection(), 48)
	if err != nil {
		log.Printf("Cannot read: %s\n", err)
		return nil, err
	}

	acceptSession := NewTwampAcceptSession(acceptBuffer)

	err = checkAcceptStatus(acceptSession.accept, "session")
	if err != nil {
		return nil, err
	}

	session := &Session{conn: c, port: acceptSession.port, config: config}

	return session, nil
}

type TwampAcceptSession struct {
	accept byte
	port   uint16
	sid    [16]byte
}

func NewTwampAcceptSession(buf bytes.Buffer) *TwampAcceptSession {
	message := &TwampAcceptSession{}
	message.accept = byte(buf.Next(1)[0])
	_ = buf.Next(1) // mbz
	message.port = binary.BigEndian.Uint16(buf.Next(2))
	copy(message.sid[:], buf.Next(16))
	return message
}
