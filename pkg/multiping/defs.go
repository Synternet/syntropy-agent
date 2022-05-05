package multiping

import "net/netip"

const (
	timeSliceLength  = 8
	trackerLength    = 8
	protocolICMP     = 1
	protocolIPv6ICMP = 58
)

type ProtocolVersion int

const (
	ProtocolIpv4 = ProtocolVersion(4)
	ProtocolIpv6 = ProtocolVersion(6)
)

var (
	ipv4Proto = map[string]string{"icmp": "ip4:icmp", "udp": "udp4"}
	ipv6Proto = map[string]string{"icmp": "ip6:ipv6-icmp", "udp": "udp6"}
)

type packet struct {
	bytes  []byte
	nbytes int
	ttl    int
	proto  ProtocolVersion
	src    netip.Addr
}
