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
	"net/netip"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent/pkg/multiping/pingdata"
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

	ctx    context.Context    // context for timeouting
	cancel context.CancelFunc // Do I need it ?

	pinger   *Pinger
	pingData *pingdata.PingData

	id       uint16
	sequence uint16 // ICMP seq number. Incremented on every ping
	network  string // one of "ip", "ip4", or "ip6"
	protocol string // protocol is "icmp" or "udp".
	conn4    *icmp.PacketConn
	conn6    *icmp.PacketConn
}

func New(privileged bool) (*MultiPing, error) {
	protocol := "udp"
	if privileged {
		protocol = "icmp"
	}

	rand.Seed(time.Now().UnixNano())
	mp := &MultiPing{
		Timeout:  time.Second,
		id:       uint16(rand.Intn(0xffff)),
		network:  "ip",
		protocol: protocol,
		Tracker:  rand.Int63(),
	}

	mp.pinger = NewPinger(mp.network, mp.protocol, mp.id)
	mp.pinger.SetPrivileged(privileged)
	mp.pinger.Tracker = mp.Tracker

	// try initialise connections to test that everything's working
	err := mp.restart()
	if err != nil {
		mp.close()
		return nil, err
	}

	// Sequence counter. It will be incremented in mp.restart on every ping
	// Start with quite big initial value, so overwrap will occure fast (easier debugin)
	mp.sequence = 0xfff0

	return mp, nil
}

func (mp *MultiPing) restart() (err error) {
	// ipv4
	mp.conn4, err = icmp.ListenPacket(ipv4Proto[mp.protocol], "")
	if err != nil {
		return err
	}
	err = mp.conn4.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)
	if err != nil {
		return err
	}

	// ipv6 (note IPv6 may be disabled on OS and may fail)
	mp.conn6, err = icmp.ListenPacket(ipv6Proto[mp.protocol], "")
	if err == nil {
		mp.conn6.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true)
	}

	mp.pinger.conn4 = mp.conn4
	mp.pinger.conn6 = mp.conn6
	mp.sequence++
	// I use zero sequence number in statistics struct
	// to detect duplicates, thus don't use it as valid sequence number
	if mp.sequence == 0 {
		mp.sequence++
	}

	return nil
}

// closes active connections
func (mp *MultiPing) close() {
	if mp.conn4 != nil {
		mp.conn4.Close()
	}
	if mp.conn6 != nil {
		mp.conn6.Close()
	}
}

// cleanup cannot be done in close, because some goroutines may be using struct members
func (mp *MultiPing) cleanup() {
	// invalidate connections
	mp.pinger.conn4 = nil
	mp.pinger.conn6 = nil
	mp.conn4 = nil
	mp.conn6 = nil

	// Invalidate pingData pointer (prevent from possible data corruption in future)
	mp.pingData = nil
	// Invalidate IP address
	mp.pinger.SetIPAddr(nil)
}

// Ping is blocking function and runs for mp.Timeout time and pings all hosts in data
func (mp *MultiPing) Ping(data *pingdata.PingData) {
	if data.Count() == 0 {
		return
	}

	// Lock the pinger - its instance may be reused by several clients
	mp.Lock()
	defer mp.Unlock()

	err := mp.restart()
	if err != nil {
		return
	}

	// Some subfunctions in goroutines will need this pointer to store ping results
	mp.pingData = data

	var wg sync.WaitGroup

	mp.ctx, mp.cancel = context.WithTimeout(context.Background(), mp.Timeout)
	defer mp.cancel()

	if mp.conn4 != nil {
		wg.Add(1)
		go mp.batchRecvICMP(&wg, ProtocolIpv4)
	}
	if mp.conn6 != nil {
		wg.Add(1)
		go mp.batchRecvICMP(&wg, ProtocolIpv6)
	}

	// Sender goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		mp.pingData.Iterate(func(addr netip.Addr, stats *pingdata.PingStats) {
			mp.pinger.SetIPAddr(&addr)
			stats.Send(mp.sequence)

			mp.pinger.SendICMP(mp.sequence)
			time.Sleep(time.Millisecond)
		})
	}()

	// wait for timeout and close connections
	<-mp.ctx.Done()
	mp.close()

	// wait for all goroutines to terminate
	wg.Wait()

	mp.cleanup()
}

func (mp *MultiPing) batchRecvICMP(wg *sync.WaitGroup, proto ProtocolVersion) {

	var packetsWait sync.WaitGroup

	defer func() {
		packetsWait.Wait()
		wg.Done()
	}()

	for {
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

		// Error reeading from connection. Can happen one of 2:
		//  * connections are closed after context timeout (most probably)
		//  * other unhandled erros (when can they happen?)
		// In either case terminate and exit
		if err != nil {
			return
		}

		// TODO: maybe there is more effective way to get netip.Addr from PacketConn ?
		var ip string
		if mp.protocol == "udp" {
			ip, _, err = net.SplitHostPort(src.String())
			if err != nil {
				continue
			}
		} else {
			ip = src.String()
		}

		var addr netip.Addr
		addr, err = netip.ParseAddr(ip)
		if err != nil {
			continue
		}

		packetsWait.Add(1)
		recv := &packet{bytes: bytes, nbytes: n, ttl: ttl, proto: proto, src: addr}
		go mp.processPacket(&packetsWait, recv)
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
		if uint16(pkt.ID) != mp.id {
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

	if stats, ok := mp.pingData.Get(recv.src); ok {
		stats.Recv(uint16(pkt.Seq), time.Since(timestamp))
	}
}
