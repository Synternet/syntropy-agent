package multiping

/**
 *    ***   The motivation for this multi-ping fork   ***
 *
 * There are quite a few Go pinger, but all of them have issues:
 *  * https://github.com/go-ping/ping works fine, but has problems when running
 *    several pingers in goroutines. When pinging ~300 hosts it looses ~1/3 packets.
 *  * https://github.com/caucy/batch_ping is umaintened for a long time and did not work for me at all.
 *  * https://github.com/rosenlo/go-MultiPing is a very young fork, has issues with logger, some parts
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

type MultiPing struct {
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

func New(privileged bool) (*MultiPing, error) {
	var err error
	protocol := "udp"
	if privileged {
		protocol = "icmp"
	}

	rand.Seed(time.Now().UnixNano())
	mp := &MultiPing{
		Timeout:  time.Second,
		id:       rand.Intn(0xffff),
		network:  "ip",
		protocol: protocol,
		done:     make(chan bool),
		pingers:  make(map[string]*Pinger),
		Tracker:  rand.Int63(),
	}

	// ipv4
	mp.conn4, err = icmp.ListenPacket(ipv4Proto[protocol], "")
	if err != nil {
		return nil, err
	}
	err = mp.conn4.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)
	if err != nil {
		return nil, err
	}

	// ipv6 (note IPv6 may be disabled on OS and may fail)
	mp.conn6, err = icmp.ListenPacket(ipv6Proto[mp.protocol], "")
	if err == nil {
		mp.conn6.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true)
	}

	return mp, nil
}

func (mp *MultiPing) reset() {
	mp.done = make(chan bool)
	mp.pingers = make(map[string]*Pinger)
	mp.sequence++
}

func (mp *MultiPing) Close() {
	if mp.conn4 != nil {
		mp.conn4.Close()
	}
	if mp.conn6 != nil {
		mp.conn6.Close()
	}
}

// Ping is blocking function and runs for mp.Timeout time and pings all hosts in data
func (mp *MultiPing) Ping(data *PingData) {
	if data.Count() == 0 {
		return
	}

	// Lock the pinger - its instance may be reused by several clients
	mp.Lock()
	defer mp.Unlock()
	mp.reset()

	// lock the results data
	data.mutex.Lock()
	defer data.mutex.Unlock()

	for host, _ := range data.entries {
		// TODO in future try optimise and get rid of this multipple pinger stuff
		pinger, err := NewPinger(host, mp.network, mp.protocol, mp.id)
		if err != nil {
			return
		}
		pinger.Tracker = mp.Tracker
		pinger.SetConns(mp.conn4, mp.conn6)
		mp.pingers[pinger.IPAddr().String()] = pinger
	}

	var wg sync.WaitGroup
	if mp.conn4 != nil {
		wg.Add(1)
		go mp.batchRecvICMP(&wg, ProtocolIpv4)
	}
	if mp.conn6 != nil {
		wg.Add(1)
		go mp.batchRecvICMP(&wg, ProtocolIpv6)
	}

	timeout := time.NewTimer(mp.Timeout)
	defer timeout.Stop()

	go mp.batchSendICMP()

	<-timeout.C
	close(mp.done)
	wg.Wait()

	// finally process results from pingers to data
	mp.processResults(data)
}

func (mp *MultiPing) batchSendICMP() {
	for _, pinger := range mp.pingers {
		pinger.SendICMP(mp.sequence)
		time.Sleep(time.Millisecond)
	}
}

func (mp *MultiPing) batchRecvICMP(wg *sync.WaitGroup, proto ProtocolVersion) {
	defer wg.Done()

	for {
		select {
		case <-mp.done:
			return
		default:
			bytes := make([]byte, 512)
			var n, ttl int
			var err error
			var src net.Addr

			if proto == ProtocolIpv4 {
				mp.conn4.SetReadDeadline(time.Now().Add(mp.Timeout))

				var cm *ipv4.ControlMessage
				n, cm, src, err = mp.conn4.IPv4PacketConn().ReadFrom(bytes)
				if cm != nil {
					ttl = cm.TTL
				}
			} else {
				mp.conn6.SetReadDeadline(time.Now().Add(mp.Timeout))

				var cm *ipv6.ControlMessage
				n, cm, src, err = mp.conn6.IPv6PacketConn().ReadFrom(bytes)
				if cm != nil {
					ttl = cm.HopLimit
				}
			}
			// Error reeading from connection
			if err != nil {
				return
			}

			recv := &packet{bytes: bytes, nbytes: n, ttl: ttl, proto: proto, src: src}
			go mp.processPacket(recv)
		}
	}
}

// This function runs in goroutine and nobody is interested in return errors
// Discard errors silently
func (mp *MultiPing) processPacket(recv *packet) {
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
	if mp.protocol == "icmp" {
		// Check if reply from same ID
		if pkt.ID != mp.id {
			return
		}
	}

	if len(pkt.Data) < timeSliceLength+trackerLength {
		return
	}

	tracker := bytesToInt(pkt.Data[timeSliceLength:])
	timestamp := bytesToTime(pkt.Data[:timeSliceLength])

	if tracker != mp.Tracker {
		return
	}

	var ip string
	if mp.protocol == "udp" {
		if ip, _, err = net.SplitHostPort(recv.src.String()); err != nil {
			return
		}
	} else {
		ip = recv.src.String()
	}

	rtt := time.Since(timestamp)

	if pinger, ok := mp.pingers[ip]; ok {
		pinger.PacketsRecv++
		pinger.rtts = append(pinger.rtts, rtt)
	}
}

func (mp *MultiPing) processResults(data *PingData) {
	// data is already locked from Ping function
	for ip, pinger := range mp.pingers {
		entry, ok := data.entries[ip]
		if !ok {
			continue
		}
		stats := pinger.Statistics()
		entry.tx = entry.tx + uint(stats.PacketsSent)
		entry.rx = entry.rx + uint(stats.PacketsRecv)
		entry.rtt = stats.AvgRtt
	}
}
