package pinger

import (
	"bytes"
	"net"
	"net/netip"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func (p *Pinger) SendICMP(addr netip.Addr, sequence uint16) error {
	pkt, err := p.PrepareICMP(addr, sequence)

	if err != nil {
		return err
	}

	return p.SendPacket(pkt)
}

func (p *Pinger) PrepareICMP(addr netip.Addr, seq uint16) (*Packet, error) {
	var err error
	pkt := Packet{
		Addr: addr,
	}

	t := append(timeToBytes(time.Now()), intToBytes(p.Tracker)...)
	if remainSize := p.Size - timeSliceLength - trackerLength; remainSize > 0 {
		t = append(t, bytes.Repeat([]byte{1}, remainSize)...)
	}

	body := &icmp.Echo{
		ID:   int(p.id), // ICMP packet's id field is uint16, not sure why Echo struct has int there
		Seq:  int(seq),  // ICMP packet's sequence field is uint16, not sure why Echo struct has int there
		Data: t,
	}

	msg := &icmp.Message{
		Code: 0,
		Body: body,
	}

	if addr.Is4() {
		msg.Type = ipv4.ICMPTypeEcho
		pkt.Proto = ProtocolIpv4
	} else {
		msg.Type = ipv6.ICMPTypeEchoRequest
		pkt.Proto = ProtocolIpv6
	}

	pkt.Bytes, err = msg.Marshal(nil)
	if err != nil {
		return nil, err
	}
	pkt.Len = len(pkt.Bytes)
	return &pkt, nil
}

func (p *Pinger) SendPacket(pkt *Packet) error {
	var err error

	var dst net.Addr
	if p.protocol == "udp" {
		dst = &net.UDPAddr{IP: pkt.Addr.AsSlice(), Zone: pkt.Addr.Zone()}
	} else {
		dst = &net.IPAddr{IP: pkt.Addr.AsSlice()}
	}

	// Some retries in case of ENOBUFS may occure
	// Do not retry infinitely
	for tries := 6; tries > 0; tries-- {
		if pkt.Proto == ProtocolIpv4 {
			if p.conn4 == nil {
				return ErrInvalidConn
			}
			_, err = p.conn4.WriteTo(pkt.Bytes, dst)
		} else {
			if p.conn6 == nil {
				return ErrInvalidConn
			}
			_, err = p.conn6.WriteTo(pkt.Bytes, dst)
		}

		if err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Err == syscall.ENOBUFS {
					continue
				}
			}
		}

		break
	}

	return err
}
