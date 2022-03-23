package twamp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"golang.org/x/net/ipv4"
)

/*
	TWAMP test connection used for running TWAMP tests.
*/
type twampTest struct {
	session *Client
	conn    *net.UDPConn
	seq     uint32
	stats   TwampStats
}

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

func (c *Client) GetStats() *TwampStats {
	return &c.test.stats
}

func (t *twampTest) setConnection(conn *net.UDPConn) error {
	c := ipv4.NewConn(conn)

	// RFC recommends IP TTL of 255
	err := c.SetTTL(255)
	if err != nil {
		return err
	}

	err = c.SetTOS(t.session.GetConfig().TOS)
	if err != nil {
		return err
	}

	t.conn = conn
	return nil
}
func (c *Client) remoteTestAddr() (*net.UDPAddr, error) {
	return net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", c.host, c.testPort))
}

/*
	Get the local IP address for the TWAMP control session.
*/
func (t *twampTest) GetLocalTestHost() string {
	localAddress := t.session.GetConnection().LocalAddr()
	return strings.Split(localAddress.String(), ":")[0]
}

/*
	Get the remote IP address for the TWAMP control session.
*/
func (t *twampTest) GetRemoteTestHost() string {
	remoteAddress := t.session.GetConnection().RemoteAddr()
	return strings.Split(remoteAddress.String(), ":")[0]
}

/*
	Run a TWAMP test and return a pointer to the TwampResults.
*/
func (c *Client) Run() (*TwampStats, error) {
	senderSeqNum := c.test.seq
	padSize := c.GetConfig().Padding

	c.test.stats.tx++
	c.test.sendTestMessage(uint(c.config.Padding), false)

	// Set timeout for test
	err := c.test.conn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		return nil, fmt.Errorf("setting test deadline: %s", err)
	}

	// receive test packets. Buffer size is TestResponce struct + padding length
	resp := new(TestResponse)
	buf := make([]byte, binary.Size(resp)+padSize)

	_, _, err = c.test.conn.ReadFrom(buf)
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
	c.test.stats.rtt = time.Now().Sub(resp.SenderTimestamp.GetTime())
	c.test.stats.avgRtt = (time.Duration(c.test.stats.rx)*c.test.stats.avgRtt + c.test.stats.rtt) / time.Duration(c.test.stats.rx+1)
	c.test.stats.rx++

	return c.GetStats(), nil
}

func (t *twampTest) sendTestMessage(padSize uint, use_all_zeroes bool) int {
	writer := new(bytes.Buffer)

	testRq := TestRequest{
		Sequence:  t.seq,
		Timestamp: NewTimestamp(time.Now()),
		ErrorEst:  1<<8 | 1, // Synchronized, MBZ, Scale + multiplier. TODO: use constants
	}
	t.seq++

	binary.Write(writer, binary.BigEndian, testRq)

	padding := make([]byte, padSize)
	if !use_all_zeroes {
		// seed psuedo-random number generator if requested
		rand.NewSource(int64(time.Now().Unix()))
		for i := 0; i < cap(padding); i++ {
			padding[i] = byte(rand.Intn(255))
		}
	}
	writer.Write(padding)

	sendMessage(t.conn, writer.Bytes())
	return writer.Len()
}
