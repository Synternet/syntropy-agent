package pinger

import (
	"fmt"
	"math/rand"
	"net/netip"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func TestPingPacket(t *testing.T) {
	ip := netip.MustParseAddr("127.0.0.1")

	var seq uint16 = 0
	for {
		seq = seq<<1 + 1
		id := uint16(rand.Intn(0xffff))

		pinger := NewPinger("ip", "icmp", id)
		if pinger == nil {
			t.Fatalf("Could not create pinger")
		}

		txPkt, err := pinger.PrepareICMP(ip, seq)
		if err != nil {
			t.Fatalf("Icmp prepare %s", err)
		}

		packet, err := icmp.ParseMessage(ProtocolICMP, txPkt.Bytes)
		if err != nil {
			t.Fatalf("Icmp parse %s", err)
		}

		data, ok := packet.Body.((*icmp.Echo))
		if !ok {
			t.Fatalf("Invalid packet body")
		}

		if data.Seq != int(seq) {
			t.Fatalf("Invalid sequence")
		}

		if data.ID != int(id) {
			t.Fatalf("Invalid ID")
		}

		timestamp := bytesToTime(data.Data[:timeSliceLength])
		tracker := bytesToInt(data.Data[timeSliceLength:])

		timeDiff := time.Since(timestamp)

		if timeDiff < 0 || timeDiff > time.Second {
			t.Fatalf("Invalid time diff %s", timeDiff.String())
		}

		if tracker != pinger.Tracker {
			t.Fatalf("Invalid tracker")
		}

		if seq == 0xffff {
			break
		}
	}
}

const testSeq = 3131

func TestSendRecv(t *testing.T) {
	p := NewPinger("ip", "udp", 111)

	conn4, err := icmp.ListenPacket("udp4", "")
	if err != nil {
		t.Fatal("UDP connection create failed")
	}
	conn4.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)

	p.SetConns(conn4, nil)
	conn4.SetReadDeadline(time.Now().Add(time.Second))

	var mainErr error
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		pkt, err := p.RecvPacket(ProtocolIpv4)
		if err != nil {
			mainErr = fmt.Errorf("Receive failed: %s", err)
			return
		}
		pingStats := p.ParsePacket(pkt)

		if pingStats.Seq != testSeq {
			mainErr = fmt.Errorf("Invalid sequence. Expected: %d received: %d", testSeq, pingStats.Seq)
			return
		}
	}()

	p.SendICMP(netip.MustParseAddr("127.0.0.1"), testSeq)

	wg.Wait()
	if mainErr != nil {
		t.Fatal(mainErr.Error())
	}
}
