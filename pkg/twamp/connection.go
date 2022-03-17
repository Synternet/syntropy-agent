package twamp

import (
	"time"
)

type SessionConfig struct {
	Port    int
	Padding int
	Timeout int
	TOS     int
}
type RequestSession struct {
	Five          byte
	IPVN          byte
	ConfSender    byte
	ConfReceiver  byte
	Slots         uint32
	Packets       uint32
	SenderPort    uint16
	ReceiverPort  uint16
	SendAddress   uint32
	SendAddress2  [12]byte
	RecvAddress   uint32
	RecvAddress2  [12]byte
	SID           [16]byte
	PaddingLength uint32
	StartTime     Timestamp
	Timeout       uint64
	TypeP         uint32
	MBZ           [8]byte
	HMAC          [16]byte
}

type AcceptSession struct {
	Accept byte
	MBZ    byte
	Port   uint16
	SID    [16]byte
	MBZ2   [12]byte
	HMAC   [16]byte
}

func (c *Client) CreateSession(config SessionConfig) (*Session, error) {
	// Send SessionRequest message
	req := new(RequestSession)
	req.Five = 5          // TODO
	req.SenderPort = 6666 // TODO: why hardcode ?
	req.ReceiverPort = uint16(config.Port)
	req.PaddingLength = uint32(config.Padding)
	req.StartTime = NewTimestamp(time.Now())
	req.Timeout = uint64(config.Timeout)
	req.TypeP = uint32(config.TOS)

	err := sendMessage(c.GetConnection(), req)
	if err != nil {
		return nil, err
	}

	// Receive AcceptSession message
	resp := new(AcceptSession)
	err = receiveMessage(c.GetConnection(), resp)
	if err != nil {
		return nil, err
	}

	err = checkAcceptStatus(resp.Accept, "session")
	if err != nil {
		return nil, err
	}

	session := &Session{conn: c, port: resp.Port, config: config}

	return session, nil
}
