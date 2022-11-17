package pinger

import (
	"net"
	"net/netip"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func (p *Pinger) RecvICMP(proto ProtocolVersion) IcmpStats {
	pkt, err := p.RecvPacket(proto)
	if err != nil {
		return IcmpStats{
			Valid: false,
		}
	}

	return p.ParsePacket(pkt)
}

func (p *Pinger) RecvPacket(proto ProtocolVersion) (*Packet, error) {
	var n, ttl int
	var err error
	var src net.Addr
	bytes := make([]byte, 512)

	if proto == ProtocolIpv4 {
		if p.conn4 == nil {
			return nil, ErrInvalidConn
		}
		var cm *ipv4.ControlMessage
		n, cm, src, err = p.conn4.IPv4PacketConn().ReadFrom(bytes)
		if cm != nil {
			ttl = cm.TTL
		}
	} else {
		if p.conn4 == nil {
			return nil, ErrInvalidConn
		}

		var cm *ipv6.ControlMessage
		n, cm, src, err = p.conn6.IPv6PacketConn().ReadFrom(bytes)
		if cm != nil {
			ttl = cm.HopLimit
		}
	}

	// Error reeading from connection. Can happen one of 2:
	//  * connections are closed after context timeout (most probably)
	//  * other unhandled erros (when can they happen?)
	// In either case terminate and exit
	if err != nil {
		return nil, ErrInvalidConn
	}

	// TODO: maybe there is more effective way to get netip.Addr from PacketConn ?
	var ip string
	if p.protocol == "udp" {
		ip, _, err = net.SplitHostPort(src.String())
		if err != nil {
			return nil, ErrInvalidAddr
		}
	} else {
		ip = src.String()
	}

	var addr netip.Addr
	addr, err = netip.ParseAddr(ip)
	if err != nil {
		return nil, ErrInvalidAddr
	}

	return &Packet{Bytes: bytes, Len: n, TTL: ttl, Proto: proto, Addr: addr}, nil
}

func (p *Pinger) ParsePacket(recv *Packet) IcmpStats {
	ret := IcmpStats{
		Valid: true,
	}

	var m *icmp.Message
	var err error
	if recv.Proto == ProtocolIpv4 {
		m, err = icmp.ParseMessage(ProtocolICMP, recv.Bytes)
	} else {
		m, err = icmp.ParseMessage(ProtocolIPv6ICMP, recv.Bytes)
	}

	if err != nil {
		ret.Valid = false
		return ret
	}

	if m.Type != ipv4.ICMPTypeEchoReply && m.Type != ipv6.ICMPTypeEchoReply {
		// Not an echo reply, ignore it
		ret.Valid = false
		return ret
	}

	pkt, ok := m.Body.(*icmp.Echo)
	if !ok {
		ret.Valid = false
		return ret
	}

	// If we are priviledged, we can match icmp.ID
	if p.protocol == "icmp" {
		// Check if reply from same ID
		if uint16(pkt.ID) != p.id {
			ret.Valid = false
			return ret
		}
	}

	if len(pkt.Data) < timeSliceLength+trackerLength {
		ret.Valid = false
		return ret
	}

	ret.Seq = uint16(pkt.Seq)
	ret.Tracker = bytesToInt(pkt.Data[timeSliceLength:])
	timestamp := bytesToTime(pkt.Data[:timeSliceLength])
	ret.RTT = time.Since(timestamp)

	return ret
}
