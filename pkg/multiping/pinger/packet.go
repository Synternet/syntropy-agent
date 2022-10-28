package pinger

import (
	"net/netip"
	"time"
)

type Packet struct {
	Proto ProtocolVersion // protocol: 4=IPv4, 6=IPv6
	Bytes []byte          // Marshaled package
	Len   int             // length of package
	TTL   int             // TTL of the packet (currently unused)
	Addr  netip.Addr      // Dest address for sending package and Src address ro received
}

type IcmpStats struct {
	Valid   bool
	RTT     time.Duration
	Tracker int64
	Seq     uint16
}
