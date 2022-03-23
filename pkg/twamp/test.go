package twamp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

type TestRequest struct {
	Sequence  uint32
	Timestamp Timestamp
	ErrorEst  uint16
}

type TestResponse struct {
	Sequence        uint32
	Timestamp       Timestamp
	ErrorEst        uint16
	MBZ             [2]byte
	RcvTimestamp    Timestamp
	SenderSequence  uint32
	SenderTimestamp Timestamp
	SenderErrorEst  uint16
	MBZ2            [2]byte
	SenderTTL       byte
}

func (c *Client) remoteTestAddr() (*net.UDPAddr, error) {
	return net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", c.host, c.testPort))
}

// Get the local IP address for the TWAMP control session.
func (c *Client) localTestHost() string {
	return strings.Split(c.controlConn.LocalAddr().String(), ":")[0]
}

// Run a TWAMP test and return a pointer to the TwampResults.
func (c *Client) Ping() (*Statistics, error) {
	senderSeqNum := c.testSequence

	c.stats.tx++
	err := c.sendTestMessage()
	if err != nil {
		return nil, fmt.Errorf("send test message: %s", err)
	}

	// Set timeout for test
	err = c.testConn.SetReadDeadline(time.Now().Add(c.config.Timeout))
	if err != nil {
		return nil, fmt.Errorf("setting test deadline: %s", err)
	}

	// receive test packets. Buffer size is TestResponce struct + padding length
	resp := new(TestResponse)
	buf := make([]byte, binary.Size(resp)+int(c.config.PaddingSize))

	_, _, err = c.testConn.ReadFrom(buf)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewBuffer(buf)
	err = binary.Read(reader, binary.BigEndian, resp)
	if err != nil {
		return nil, err
	}
	if senderSeqNum != resp.SenderSequence {
		return nil, fmt.Errorf("expected seq %d but received %d", senderSeqNum, resp.SenderSequence)
	}

	// Successfully received and parsed message - increase rx stats
	c.stats.rtt = time.Now().Sub(resp.SenderTimestamp.GetTime())
	c.stats.avgRtt = (time.Duration(c.stats.rx)*c.stats.avgRtt + c.stats.rtt) / time.Duration(c.stats.rx+1)
	c.stats.rx++

	return c.Stats(), nil
}

func (c *Client) sendTestMessage() error {
	writer := new(bytes.Buffer)

	testRq := TestRequest{
		Sequence:  c.testSequence,
		Timestamp: NewTimestamp(time.Now()),
		ErrorEst:  1<<8 | 1, // Synchronized, MBZ, Scale + multiplier. TODO: use constants
	}
	c.testSequence++

	err := binary.Write(writer, binary.BigEndian, testRq)
	if err != nil {
		return err
	}

	padding := make([]byte, c.config.PaddingSize)
	if !c.config.PaddingZeroes {
		// seed psuedo-random number generator if requested
		rand.NewSource(int64(time.Now().Unix()))
		for i := 0; i < cap(padding); i++ {
			padding[i] = byte(rand.Intn(255))
		}
	}

	_, err = c.testConn.Write(append(writer.Bytes(), padding...))

	return err
}
