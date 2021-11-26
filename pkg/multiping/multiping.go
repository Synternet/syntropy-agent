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
	"context"
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

	ctx      context.Context // context for timeouting
	pinger   *Pinger
	pingData *PingData

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
		Tracker:  rand.Int63(),
	}

	mp.pinger = NewPinger(mp.network, mp.protocol, mp.id)
	mp.pinger.SetPrivileged(privileged)
	mp.pinger.Tracker = mp.Tracker

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

	mp.pinger.SetConns(mp.conn4, mp.conn6)

	return mp, nil
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
	mp.sequence++

	// lock the results data
	data.mutex.Lock()
	defer data.mutex.Unlock()

	// Some subfunctions in goroutines will need this pointer to store ping results
	mp.pingData = data
	ipAddrs := []*net.IPAddr{}

	// TODO when GO1.18 will have netip struct, use netip address as index instead of string
	// And remove this address resolve
	for host, stats := range mp.pingData.entries {
		ip, err := net.ResolveIPAddr(mp.network, host)
		if err != nil {
			// ResolveIP failed. I should not return here, so instead I increment Tx packet count
			// And this (invalid) host will result to ping loss.
			// Its up to caller to pass me valid addresses
			stats.tx++
			stats.rtt = 0
			continue
		}
		ipAddrs = append(ipAddrs, ip)
	}

	var wg sync.WaitGroup
	wg.Add(1) // Sender goroutine

	mp.ctx, _ = context.WithTimeout(context.Background(), mp.Timeout)

	if mp.conn4 != nil {
		wg.Add(1)
		go mp.batchRecvICMP(&wg, ProtocolIpv4)
	}
	if mp.conn6 != nil {
		wg.Add(1)
		go mp.batchRecvICMP(&wg, ProtocolIpv6)
	}

	// Sender goroutine
	go func() {
		defer wg.Done()
		for _, addr := range ipAddrs {
			mp.pinger.SetIPAddr(addr)
			if stats, ok := mp.pingData.entries[addr.IP.String()]; ok {
				stats.tx++
			}

			mp.pinger.SendICMP(mp.sequence)
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()

	// Invalidate pingData pointer (prevent from possible data corruption in future)
	mp.pingData = nil
}

func (mp *MultiPing) batchRecvICMP(wg *sync.WaitGroup, proto ProtocolVersion) {
	defer wg.Done()

	var packetsWait sync.WaitGroup

	for {
		select {
		case <-mp.ctx.Done():
			packetsWait.Wait()
			return
		default:
			bytes := make([]byte, 512)
			var n, ttl int
			var err error
			var src net.Addr

			if proto == ProtocolIpv4 {
				mp.conn4.SetReadDeadline(time.Now().Add(mp.Timeout / 10))

				var cm *ipv4.ControlMessage
				n, cm, src, err = mp.conn4.IPv4PacketConn().ReadFrom(bytes)
				if cm != nil {
					ttl = cm.TTL
				}
			} else {
				mp.conn6.SetReadDeadline(time.Now().Add(mp.Timeout / 10))

				var cm *ipv6.ControlMessage
				n, cm, src, err = mp.conn6.IPv6PacketConn().ReadFrom(bytes)
				if cm != nil {
					ttl = cm.HopLimit
				}
			}
			// Error reeading from connection
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						continue
					} else {
						return
					}
				}
			}

			packetsWait.Add(1)
			recv := &packet{bytes: bytes, nbytes: n, ttl: ttl, proto: proto, src: src}
			go mp.processPacket(&packetsWait, recv)
		}
	}
}

// This function runs in goroutine and nobody is interested in return errors
// Discard errors silently
func (mp *MultiPing) processPacket(wait *sync.WaitGroup, recv *packet) {
	defer wait.Done()

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

	if stats, ok := mp.pingData.entries[ip]; ok {
		stats.rx++
		stats.rtt = time.Since(timestamp)
	}
}
