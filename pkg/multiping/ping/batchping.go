package ping

/**
 *    ***   The motivation for this multi-ping fork   ***
 *
 * There are quite a few Go pinger, but all of them have issues:
 *  * https://github.com/go-ping/ping works fine, but has problems when running
 *    several pingers in goroutines. When pinging ~300 hosts it looses ~1/3 packets.
 *  * https://github.com/caucy/batch_ping is umaintened for a long time and did not work for me at all.
 *  * https://github.com/rosenlo/go-batchping is a very young fork, has issues with logger, some parts
 *    of code are ineffective.
 *
 *  Also need to mention that all these pingers are periodic pingers, they try to mimmic shell ping command.
 * They run in internal loop, cancel that loop after timeout. They *can* be used, but you have to adjust your
 * code to their style. Instead I wanted a pinger, that can ping multipple hosts at a time and be robust.
 * I don't think its a problem for ping user to run it in a loop and don't want any hidden logic.
 * So this ping is loosely based on above mentioned projects. It can ping multipple clients.
 * And is cholesterol free.
 **/

import (
	"math/rand"
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type BatchPing struct {
	sync.RWMutex

	// Timeout specifies a timeout before ping exits, regardless of how many
	// packets have been received. Default is 1s.
	Timeout time.Duration

	// Tracker: Used to uniquely identify packet when non-priviledged
	Tracker int64

	done    chan bool          // close channel
	pingers map[string]*Pinger // TODO in future try to get rid of multipple pingers

	id       int
	sequence int    // ICMP seq number. Incremented on every ping
	network  string // one of "ip", "ip4", or "ip6"
	protocol string // protocol is "icmp" or "udp".
	conn4    *icmp.PacketConn
	conn6    *icmp.PacketConn
}

func New(privileged bool) (*BatchPing, error) {
	var err error
	protocol := "udp"
	if privileged {
		protocol = "icmp"
	}

	rand.Seed(time.Now().UnixNano())
	bp := &BatchPing{
		Timeout:  time.Second,
		id:       rand.Intn(0xffff),
		network:  "ip",
		protocol: protocol,
		done:     make(chan bool),
		pingers:  make(map[string]*Pinger),
		Tracker:  rand.Int63(),
	}

	// ipv4
	bp.conn4, err = icmp.ListenPacket(ipv4Proto[protocol], "")
	if err != nil {
		return nil, err
	}
	err = bp.conn4.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)
	if err != nil {
		return nil, err
	}

	// ipv6 (note IPv6 may be disabled on OS and may fail)
	bp.conn6, err = icmp.ListenPacket(ipv6Proto[bp.protocol], "")
	if err == nil {
		bp.conn6.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true)
	}

	return bp, nil
}

func (bp *BatchPing) reset() {
	bp.done = make(chan bool)
	bp.pingers = make(map[string]*Pinger)
	bp.sequence++
}

func (bp *BatchPing) Close() {
	if bp.conn4 != nil {
		bp.conn4.Close()
	}
	if bp.conn6 != nil {
		bp.conn6.Close()
	}
}

func (bp *BatchPing) Run(addrs []string) {
	if len(addrs) == 0 {
		return
	}

	bp.Lock()
	defer bp.Unlock()
	bp.reset()

	for _, addr := range addrs {
		pinger, err := NewPinger(addr, bp.network, bp.protocol, bp.id)
		if err != nil {
			continue
		}
		pinger.Tracker = bp.Tracker
		pinger.SetConns(bp.conn4, bp.conn6)
		bp.pingers[pinger.IPAddr().String()] = pinger
	}

	var wg sync.WaitGroup
	if bp.conn4 != nil {
		wg.Add(1)
		go bp.batchRecvICMP(&wg, ProtocolIpv4)
	}
	if bp.conn6 != nil {
		wg.Add(1)
		go bp.batchRecvICMP(&wg, ProtocolIpv6)
	}

	timeout := time.NewTimer(bp.Timeout)
	defer timeout.Stop()

	go bp.batchSendICMP()

	<-timeout.C
	close(bp.done)
	wg.Wait()
}

func (bp *BatchPing) batchSendICMP() {
	for _, pinger := range bp.pingers {
		pinger.SendICMP(bp.sequence)
		time.Sleep(time.Millisecond)
	}
}

func (bp *BatchPing) batchRecvICMP(wg *sync.WaitGroup, proto ProtocolVersion) {
	defer wg.Done()

	for {
		select {
		case <-bp.done:
			return
		default:
			bytes := make([]byte, 512)
			var n, ttl int
			var err error
			var src net.Addr

			if proto == ProtocolIpv4 {
				bp.conn4.SetReadDeadline(time.Now().Add(bp.Timeout))

				var cm *ipv4.ControlMessage
				n, cm, src, err = bp.conn4.IPv4PacketConn().ReadFrom(bytes)
				if cm != nil {
					ttl = cm.TTL
				}
			} else {
				bp.conn6.SetReadDeadline(time.Now().Add(bp.Timeout))

				var cm *ipv6.ControlMessage
				n, cm, src, err = bp.conn6.IPv6PacketConn().ReadFrom(bytes)
				if cm != nil {
					ttl = cm.HopLimit
				}
			}
			// Error reeading from connection
			if err != nil {
				return
			}

			recv := &packet{bytes: bytes, nbytes: n, ttl: ttl, proto: proto, src: src}
			go bp.processPacket(recv)
		}
	}
}

// This function runs in goroutine and nobody is interested in return errors
// Discard errors silently
func (bp *BatchPing) processPacket(recv *packet) {
	var proto int
	if recv.proto == ProtocolIpv4 {
		proto = protocolICMP
	} else {
		proto = protocolIPv6ICMP
	}

	var m *icmp.Message
	var err error
	if m, err = icmp.ParseMessage(proto, recv.bytes); err != nil {
		return
	}

	if m.Type != ipv4.ICMPTypeEchoReply && m.Type != ipv6.ICMPTypeEchoReply {
		// Not an echo reply, ignore it
		return
	}

	pkt, ok := m.Body.(*icmp.Echo)
	if !ok {
		return
	}

	// If we are priviledged, we can match icmp.ID
	if bp.protocol == "icmp" {
		// Check if reply from same ID
		if pkt.ID != bp.id {
			return
		}
	}

	if len(pkt.Data) < timeSliceLength+trackerLength {
		return
	}

	tracker := bytesToInt(pkt.Data[timeSliceLength:])
	timestamp := bytesToTime(pkt.Data[:timeSliceLength])

	if tracker != bp.Tracker {
		return
	}

	var ip string
	if bp.protocol == "udp" {
		if ip, _, err = net.SplitHostPort(recv.src.String()); err != nil {
			return
		}
	} else {
		ip = recv.src.String()
	}

	rtt := time.Since(timestamp)

	if pinger, ok := bp.pingers[ip]; ok {
		pinger.PacketsRecv++
		pinger.rtts = append(pinger.rtts, rtt)
	}
}

func (bp *BatchPing) Statistics() map[string]*Statistics {
	pingerStat := map[string]*Statistics{}
	for ip, pinger := range bp.pingers {
		pingerStat[ip] = pinger.Statistics()
	}
	return pingerStat
}
