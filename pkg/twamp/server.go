package twamp

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"syscall"
	"time"

	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

type Server struct {
	listen   string
	udpStart uint16
	sock     net.Listener
}

func NewServer(address string, startPort uint16) (*Server, error) {
	s := Server{
		listen:   fmt.Sprintf("%s:%d", address, TwampControlPort),
		udpStart: startPort,
	}

	return &s, nil
}

func (s *Server) Run() error {
	var udp_port = s.udpStart
	var err error

	s.sock, err = net.Listen("tcp", s.listen)
	if err != nil {
		return fmt.Errorf("error listening on %s: %s", s.listen, err)
	}

	for {
		conn, err := s.sock.Accept()
		if err != nil {
			return fmt.Errorf("error accepting connection: %s", err)
		}

		go handleClient(conn, udp_port)
		udp_port++
	}
}

func (s *Server) Close() error {
	if s.sock == nil {
		return fmt.Errorf("server not running")
	}
	defer func() { s.sock = nil }()

	return s.sock.Close()
}

type SetupResponse struct {
	Mode     uint32
	KeyID    [80]byte
	Token    [64]byte
	ClientIV [16]byte
}

func handleClient(conn net.Conn, udp_port uint16) {
	logger.Info().Println(pkgName, "Handle client on port", udp_port)
	err := serveClient(conn, udp_port)
	if err != nil {
		logger.Info().Println(pkgName, "server handle client error", err)
	}
}

func serveClient(conn net.Conn, udp_port uint16) error {
	defer conn.Close()

	logger.Info().Println(pkgName, "Handling control connection from client", conn.RemoteAddr())

	err := sendServerGreeting(conn)
	if err != nil {
		return fmt.Errorf("sending greeting: %s", err)
	}

	_, err = receiveSetupResponse(conn)
	if err != nil {
		return fmt.Errorf("receiving setup: %s", err)
	}

	err = sendServerStart(conn)
	if err != nil {
		return fmt.Errorf("sending start: %s", err)
	}

	_, err = receiveRequestSession(conn)
	if err != nil {
		return fmt.Errorf("receiving session: %s", err)
	}

	udp_conn, err := startReflector(udp_port)
	if err != nil {
		return fmt.Errorf("starting reflector on port %d: %s", udp_port, err)
	}

	err = sendAcceptSession(conn, udp_port)
	if err != nil {
		return fmt.Errorf("sending session accept: %s", err)
	}

	_, err = receiveStartSessions(conn)
	if err != nil {
		return fmt.Errorf("receiving start sessions: %s", err)
	}

	test_done := make(chan bool)
	defer close(test_done)
	go handleReflector(udp_conn, test_done)

	err = sendStartAck(conn)
	if err != nil {
		return fmt.Errorf("sending start ACK: %s", err)
	}

	_, err = receiveStopSessions(conn)
	if err != nil {
		return fmt.Errorf("receiving stop sessions: %s", err)
	}

	logger.Info().Println(pkgName, "Finished control connection from client", conn.RemoteAddr())
	return nil
}

func receiveSetupResponse(conn net.Conn) (*SetupResponse, error) {
	setup := new(SetupResponse)

	err := receiveMessage(conn, setup)
	if err != nil {
		return nil, err
	}

	if setup.Mode != ModeUnauthenticated {
		err = fmt.Errorf("unsupported setup mode received %d", setup.Mode)
		return nil, err
	}

	return setup, nil
}

func receiveRequestSession(conn net.Conn) (*RequestSession, error) {
	session := new(RequestSession)

	err := receiveMessage(conn, session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func receiveStartSessions(conn net.Conn) (*StartSessions, error) {
	msg := new(StartSessions)

	err := receiveMessage(conn, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func receiveStopSessions(conn net.Conn) (*StopSessions, error) {
	msg := new(StopSessions)

	err := receiveMessage(conn, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func createServerStart(accept byte) (*ServerStart, error) {
	start := new(ServerStart)

	start.Accept = accept

	ts := NewTimestamp(time.Now())
	start.StartTime.Seconds = ts.Seconds
	start.StartTime.Fraction = ts.Fraction

	_, err := rand.Read(start.ServerIV[:])
	if err != nil {
		return nil, err
	}

	return start, nil
}

func sendServerStart(conn net.Conn) error {
	start, err := createServerStart(AcceptOK)
	if err != nil {
		return err
	}

	err = sendMessage(conn, start)
	if err != nil {
		return err
	}

	return nil
}

func createAcceptSession(accept byte, port uint16) (*AcceptSession, error) {
	msg := new(AcceptSession)

	msg.Accept = accept
	msg.Port = port
	_, err := rand.Read(msg.SID[:])
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func sendAcceptSession(conn net.Conn, udp_port uint16) error {
	msg, err := createAcceptSession(AcceptOK, udp_port)
	if err != nil {
		return err
	}

	err = sendMessage(conn, msg)
	if err != nil {
		return err
	}

	return nil
}

func createStartAck(accept byte) (*StartAck, error) {
	msg := new(StartAck)

	msg.Accept = accept

	return msg, nil
}

func sendStartAck(conn net.Conn) error {
	msg, err := createStartAck(AcceptOK)
	if err != nil {
		return err
	}

	err = sendMessage(conn, msg)
	if err != nil {
		return err
	}

	return nil
}

func createTestResponse(buf []byte, seq uint32) ([]byte, error) {
	req := new(TestRequest)
	reader := bytes.NewBuffer(buf)
	err := binary.Read(reader, binary.BigEndian, req)
	if err != nil {
		return nil, err
	}
	received := time.Now()

	resp := new(TestResponse)
	resp.SenderSequence = req.Sequence
	resp.SenderTimestamp = req.Timestamp
	resp.SenderErrorEst = req.ErrorEst
	/* TODO
	   -  Extract the Sender TTL value from the TTL/Hop Limit value of
	      received packets.  Session-Reflector implementations SHOULD fetch
	      the TTL/Hop Limit value from the IP header of the packet,
	      replacing the value of 255 set by the Session-Sender.  If an
	      implementation does not fetch the actual TTL value (the only good
	      reason not to do so is an inability to access the TTL field of
	      arriving packets), it MUST set the Sender TTL value as 255.
	*/
	resp.SenderTTL = 255

	resp.Sequence = seq
	resp.RcvTimestamp = NewTimestamp(received)
	resp.ErrorEst = createErrorEstimate()

	writer := new(bytes.Buffer)
	resp.Timestamp = NewTimestamp(time.Now())
	err = binary.Write(writer, binary.BigEndian, resp)
	if err != nil {
		return nil, err
	}

	// Copy same padding to reply back
	_, err = writer.Write(buf[binary.Size(req):])
	if err != nil {
		return nil, err
	}

	return writer.Bytes(), nil
}

func createErrorEstimate() uint16 {
	var estimate uint16 = 0x3FFF

	var buf syscall.Timex
	_, err := syscall.Adjtimex(&buf)
	if err != nil {
		return estimate
	}

	multiplier := buf.Esterror
	multiplier <<= 32
	multiplier /= 1000000

	var scale uint16
	for multiplier >= 0xFF {
		scale++
		multiplier >>= 1
	}

	estimate = 1 << 15
	estimate |= scale << 8
	estimate |= uint16(multiplier & 0xFF)

	return estimate
}

func startReflector(udp_port uint16) (*net.UDPConn, error) {
	listen := ":" + strconv.Itoa(int(udp_port))
	laddr, err := net.ResolveUDPAddr("udp", listen)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func handleReflector(conn *net.UDPConn, test_done chan bool) {
	err := runReflector(conn, test_done)
	if err != nil {
		logger.Error().Println(pkgName, "reflector error:", err)
	}
}

func runReflector(conn *net.UDPConn, test_done chan bool) error {
	var seq uint32 = 0
	buf := make([]byte, 10240)
	timeout := 10 * time.Second
	defer conn.Close()

	logger.Info().Println(pkgName, "Handling test session on port", conn.LocalAddr())
	for {
		err := conn.SetReadDeadline(time.Now().Add(timeout))
		if err != nil {
			return fmt.Errorf("setting test deadline: %s", err)
		}

		pktLen, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				if _, ok := <-test_done; !ok {
					logger.Info().Println(pkgName, "Finished test session on port", conn.LocalAddr())
					return nil
				} else {
					logger.Info().Println(pkgName, "Timeout waiting for test packet:", err)
					continue
				}
			}

			return fmt.Errorf("receiving test packet: %s", err)
		}

		response, err := createTestResponse(buf[:pktLen], seq)
		if err != nil {
			return fmt.Errorf("creating test response: %s", err)
		}

		_, err = conn.WriteToUDP(response, addr)
		if err != nil {
			return fmt.Errorf("sending test reponse: %s", err)
		}

		seq++
	}
}