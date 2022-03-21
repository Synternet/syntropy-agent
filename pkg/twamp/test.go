package twamp

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"golang.org/x/net/ipv4"
)

/*
	TWAMP test connection used for running TWAMP tests.
*/
type TwampTest struct {
	session *Session
	conn    *net.UDPConn
	seq     uint32
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

/*

 */
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
func (t *TwampTest) Run() (*TwampResults, error) {
	senderSeqNum := t.seq
	padSize := t.GetSession().GetConfig().Padding

	size := t.sendTestMessage(false)

	// receive test packets
	resp := new(TestResponse)
	buf := make([]byte, binary.Size(resp)+padSize)

	_, _, err := t.GetConnection().ReadFrom(buf)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewBuffer(buf)
	err = binary.Read(reader, binary.BigEndian, resp)
	if err != nil {
		return nil, err
	}

	// process test results
	r := &TwampResults{}
	r.SenderSize = size
	r.SeqNum = resp.Sequence
	r.Timestamp = ConvertTimestamp(resp.Timestamp.Seconds, resp.Timestamp.Fraction)
	r.ErrorEstimate = resp.ErrorEst
	r.ReceiveTimestamp = ConvertTimestamp(resp.RcvTimestamp.Seconds, resp.RcvTimestamp.Fraction)
	r.SenderSeqNum = resp.SenderSequence
	r.SenderTimestamp = ConvertTimestamp(resp.SenderTimestamp.Seconds, resp.SenderTimestamp.Fraction)
	r.SenderErrorEstimate = resp.SenderErrorEst
	r.SenderTTL = resp.SenderTTL
	r.FinishedTimestamp = time.Now()

	if senderSeqNum != r.SeqNum {
		return nil, fmt.Errorf("expected seq %d but received %d", senderSeqNum, r.SeqNum)
	}

	return r, nil
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

func (t *TwampTest) FormatJSON(r *PingResults) {
	doc, err := json.Marshal(r)
	if err != nil {
		logger.Error().Println(pkgName, "JSON marshal", err)
		return
	}
	fmt.Printf("%s\n", string(doc))
}

func (t *TwampTest) Ping(count int, isRapid bool, interval int) *PingResults {
	Stats := &PingResultStats{}
	Results := &PingResults{Stat: Stats}
	var TotalRTT time.Duration = 0

	packetSize := 14 + t.GetSession().GetConfig().Padding

	fmt.Printf("TWAMP PING %s: %d data bytes\n", t.GetRemoteTestHost(), packetSize)

	for i := 0; i < count; i++ {
		Stats.Transmitted++
		results, err := t.Run()
		if err != nil {
			if isRapid {
				fmt.Printf(".")
			}
		} else {
			if i == 0 {
				Stats.Min = results.GetRTT()
				Stats.Max = results.GetRTT()
			}
			if Stats.Min > results.GetRTT() {
				Stats.Min = results.GetRTT()
			}
			if Stats.Max < results.GetRTT() {
				Stats.Max = results.GetRTT()
			}

			TotalRTT += results.GetRTT()
			Stats.Received++
			Results.Results = append(Results.Results, results)

			if isRapid {
				fmt.Printf("!")
			} else {
				fmt.Printf("%d bytes from %s: twamp_seq=%d ttl=%d time=%0.03f ms\n",
					packetSize,
					t.GetRemoteTestHost(),
					results.SenderSeqNum,
					results.SenderTTL,
					(float64(results.GetRTT()) / float64(time.Millisecond)),
				)
			}
		}

		if !isRapid {
			time.Sleep(time.Duration(interval) * time.Second)
		}
	}

	if isRapid {
		fmt.Printf("\n")
	}

	Stats.Avg = time.Duration(int64(TotalRTT) / int64(count))
	Stats.Loss = float64(float64(Stats.Transmitted-Stats.Received)/float64(Stats.Transmitted)) * 100.0
	Stats.StdDev = Results.stdDev(Stats.Avg)

	fmt.Printf("--- %s twamp ping statistics ---\n", t.GetRemoteTestHost())
	fmt.Printf("%d packets transmitted, %d packets received, %0.1f%% packet loss\n",
		Stats.Transmitted,
		Stats.Received,
		Stats.Loss)
	fmt.Printf("round-trip min/avg/max/stddev = %0.3f/%0.3f/%0.3f/%0.3f ms\n",
		(float64(Stats.Min) / float64(time.Millisecond)),
		(float64(Stats.Avg) / float64(time.Millisecond)),
		(float64(Stats.Max) / float64(time.Millisecond)),
		(float64(Stats.StdDev) / float64(time.Millisecond)),
	)

	return Results
}

func (t *TwampTest) RunX(count int) *PingResults {
	Stats := &PingResultStats{}
	Results := &PingResults{Stat: Stats}
	var TotalRTT time.Duration = 0

	for i := 0; i < count; i++ {
		Stats.Transmitted++
		results, err := t.Run()
		if err != nil {
		} else {
			if i == 0 {
				Stats.Min = results.GetRTT()
				Stats.Max = results.GetRTT()
			}
			if Stats.Min > results.GetRTT() {
				Stats.Min = results.GetRTT()
			}
			if Stats.Max < results.GetRTT() {
				Stats.Max = results.GetRTT()
			}

			TotalRTT += results.GetRTT()
			Stats.Received++
			Results.Results = append(Results.Results, results)
		}
	}

	Stats.Avg = time.Duration(int64(TotalRTT) / int64(count))
	Stats.Loss = float64(float64(Stats.Transmitted-Stats.Received)/float64(Stats.Transmitted)) * 100.0
	Stats.StdDev = Results.stdDev(Stats.Avg)

	return Results
}
