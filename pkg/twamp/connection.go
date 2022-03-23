package twamp

import (
	"time"
)

type RequestSession struct {
	Command       byte
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

func (c *Client) createSession() error {
	// Send SessionRequest message
	req := new(RequestSession)
	req.Command = CmdRequestTwSession
	req.SenderPort = uint16(c.config.LocalPort)
	req.ReceiverPort = 0
	req.PaddingLength = uint32(c.config.PaddingSize)
	req.StartTime = NewTimestamp(time.Now())
	req.Timeout = uint64(c.config.Timeout)
	req.TypeP = uint32(c.config.TOS)

	err := sendMessage(c.controlConn, req)
	if err != nil {
		return err
	}

	// Receive AcceptSession message
	resp := new(AcceptSession)
	err = receiveMessage(c.controlConn, resp)
	if err != nil {
		return err
	}

	err = checkAcceptStatus(resp.Accept, "session")
	if err != nil {
		return err
	}

	c.testPort = resp.Port

	return nil
}
