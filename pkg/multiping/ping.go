// This file code is based on https://github.com/go-ping/ping
package multiping

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"net/netip"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

var errInvalidIpAddr = errors.New("invalid ip address")

// TODO think about one of:
//  * move pinger to separate package
//  * make all functions private

// NewPinger returns a new Pinger instance
func NewPinger(network, protocol string, id uint16) *Pinger {
	p := &Pinger{
		Size: timeSliceLength,

		id:       id,
		ipaddr:   nil,
		network:  network,
		protocol: protocol,
	}
	return p
}

// Pinger represents a packet sender.
type Pinger struct {
	// Size of packet being sent
	Size int

	// Tracker: Used to uniquely identify packet when non-priviledged
	Tracker int64

	ipaddr *netip.Addr

	id uint16
	// network is one of "ip", "ip4", or "ip6".
	network string
	// protocol is "icmp" or "udp".
	protocol string

	//conn4 is ipv4 icmp PacketConn
	conn4 *icmp.PacketConn

	//conn6 is ipv6 icmp PacketConn
	conn6 *icmp.PacketConn
}

// SetIPAddr sets the ip address of the target host.
func (p *Pinger) SetIPAddr(ipaddr *netip.Addr) {
	p.ipaddr = ipaddr
}

// IPAddr returns the ip address of the target host.
func (p *Pinger) IPAddr() *netip.Addr {
	return p.ipaddr
}

// Addr returns the string ip address of the target host.
func (p *Pinger) Addr() string {
	if p.ipaddr != nil {
		return p.ipaddr.String()
	}
	return ""
}

// SetPrivileged sets the type of ping pinger will send.
// false means pinger will send an "unprivileged" UDP ping.
// true means pinger will send a "privileged" raw ICMP ping.
// NOTE: setting to true requires that it be run with super-user privileges.
func (p *Pinger) SetPrivileged(privileged bool) {
	if privileged {
		p.protocol = "icmp"
	} else {
		p.protocol = "udp"
	}
}

// Privileged returns whether pinger is running in privileged mode.
func (p *Pinger) Privileged() bool {
	return p.protocol == "icmp"
}

func (p *Pinger) SendICMP(sequence uint16) error {
	var dst net.Addr
	if p.protocol == "udp" {
		dst = &net.UDPAddr{IP: p.ipaddr.AsSlice(), Zone: p.ipaddr.Zone()}
	} else {
		dst = &net.IPAddr{IP: p.ipaddr.AsSlice()}
	}

	msgBytes, err := p.prepareICMP(sequence)
	if err != nil {
		return err
	}

	return p.sendICMP(msgBytes, dst)
}

func (p *Pinger) prepareICMP(sequence uint16) ([]byte, error) {
	if p.ipaddr == nil {
		return nil, errInvalidIpAddr
	}
	var typ icmp.Type
	if p.ipaddr.Is4() {
		typ = ipv4.ICMPTypeEcho
	} else {
		typ = ipv6.ICMPTypeEchoRequest
	}

	t := append(timeToBytes(time.Now()), intToBytes(p.Tracker)...)
	if remainSize := p.Size - timeSliceLength - trackerLength; remainSize > 0 {
		t = append(t, bytes.Repeat([]byte{1}, remainSize)...)
	}

	body := &icmp.Echo{
		ID:   int(p.id),     // ICMP packet's id field is uint16, not sure why Echo struct has int there
		Seq:  int(sequence), // ICMP packet's sequence field is uint16, not sure why Echo struct has int there
		Data: t,
	}

	msg := &icmp.Message{
		Type: typ,
		Code: 0,
		Body: body,
	}

	return msg.Marshal(nil)
}

func (p *Pinger) sendICMP(msgBytes []byte, dst net.Addr) error {
	var err error
	for {
		if p.ipaddr.Is4() {
			_, err = p.conn4.WriteTo(msgBytes, dst)
		} else {
			_, err = p.conn6.WriteTo(msgBytes, dst)
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

func bytesToTime(b []byte) time.Time {
	var nsec int64
	for i := uint8(0); i < 8; i++ {
		nsec += int64(b[i]) << ((7 - i) * 8)
	}
	return time.Unix(nsec/1000000000, nsec%1000000000)
}

func isIPv4(ip net.IP) bool {
	return len(ip.To4()) == net.IPv4len
}

func timeToBytes(t time.Time) []byte {
	nsec := t.UnixNano()
	b := make([]byte, 8)
	for i := uint8(0); i < 8; i++ {
		b[i] = byte((nsec >> ((7 - i) * 8)) & 0xff)
	}
	return b
}

func bytesToInt(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

func intToBytes(tracker int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(tracker))
	return b
}
