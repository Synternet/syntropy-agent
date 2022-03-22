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
type TwampTest struct {
	session *Session
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

func (t *TwampTest) GetStats() *TwampStats {
	return &t.stats
}

func (t *TwampTest) SetConnection(conn *net.UDPConn) error {
	c := ipv4.NewConn(conn)

	// RFC recommends IP TTL of 255
	err := c.SetTTL(255)
	if err != nil {
		return err
	}

	err = c.SetTOS(t.GetSession().GetConfig().TOS)
	if err != nil {
		return err
	}

	t.conn = conn
	return nil
}

/*
	Get TWAMP Test UDP connection.
*/
func (t *TwampTest) GetConnection() *net.UDPConn {
	return t.conn
}

/*
	Get the underlying TWAMP control session for the TWAMP test.
*/
func (t *TwampTest) GetSession() *Session {
	return t.session
}

/*
	Get the remote TWAMP IP/UDP address.
*/
func (t *TwampTest) RemoteAddr() (*net.UDPAddr, error) {
	address := fmt.Sprintf("%s:%d", t.GetRemoteTestHost(), t.GetRemoteTestPort())
	return net.ResolveUDPAddr("udp", address)
}

/*
	Get the remote TWAMP UDP port number.
*/
func (t *TwampTest) GetRemoteTestPort() uint16 {
	return t.GetSession().port
}

/*
	Get the local IP address for the TWAMP control session.
*/
func (t *TwampTest) GetLocalTestHost() string {
	localAddress := t.session.GetConnection().LocalAddr()
	return strings.Split(localAddress.String(), ":")[0]
}

/*
	Get the remote IP address for the TWAMP control session.
*/
func (t *TwampTest) GetRemoteTestHost() string {
	remoteAddress := t.session.GetConnection().RemoteAddr()
	return strings.Split(remoteAddress.String(), ":")[0]
}

/*
	Run a TWAMP test and return a pointer to the TwampResults.
*/
func (t *TwampTest) Run() (*TwampStats, error) {
	senderSeqNum := t.seq
	padSize := t.GetSession().GetConfig().Padding

	t.stats.tx++
	t.sendTestMessage(false)

	// Set timeout for test
	err := t.GetConnection().SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		return nil, fmt.Errorf("setting test deadline: %s", err)
	}

	// receive test packets. Buffer size is TestResponce struct + padding length
	resp := new(TestResponse)
	buf := make([]byte, binary.Size(resp)+padSize)

	_, _, err = t.GetConnection().ReadFrom(buf)
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
	t.stats.rtt = time.Now().Sub(resp.SenderTimestamp.GetTime())
	t.stats.avgRtt = (time.Duration(t.stats.rx)*t.stats.avgRtt + t.stats.rtt) / time.Duration(t.stats.rx+1)
	t.stats.rx++

	return t.GetStats(), nil
}

func (t *TwampTest) sendTestMessage(use_all_zeroes bool) int {
	writer := new(bytes.Buffer)

	testRq := TestRequest{
		Sequence:  t.seq,
		Timestamp: NewTimestamp(time.Now()),
		ErrorEst:  1<<8 | 1, // Synchronized, MBZ, Scale + multiplier. TODO: use constants
	}
	t.seq++

	binary.Write(writer, binary.BigEndian, testRq)

	padding := make([]byte, t.GetSession().config.Padding)
	if !use_all_zeroes {
		// seed psuedo-random number generator if requested
		rand.NewSource(int64(time.Now().Unix()))
		for i := 0; i < cap(padding); i++ {
			padding[i] = byte(rand.Intn(255))
		}
	}
	writer.Write(padding)

	sendMessage(t.GetConnection(), writer.Bytes())
	return writer.Len()
}
