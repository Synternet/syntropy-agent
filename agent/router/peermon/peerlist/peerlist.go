package peerlist

import (
	"net/netip"
)

type PeerList struct {
	avgCount uint
	peers    map[netip.Prefix]*PeerInfo
}

func NewPeerList(count uint) *PeerList {
	return &PeerList{
		avgCount: count,
		peers:    make(map[netip.Prefix]*PeerInfo),
	}
}

func (pl *PeerList) AddPeer(ifname, pubKey string, endpoint netip.Prefix, connID int, disabled bool) {
	e, ok := pl.peers[endpoint]
	if !ok {
		e = NewPeerInfo(uint(pl.avgCount))
		pl.peers[endpoint] = e
	}

	e.Ifname = ifname
	e.PublicKey = pubKey
	e.ConnectionID = connID
	e.flags |= PifAddPending
	e.flags &= ^PifDelPending
	if disabled {
		e.flags |= PifDisabled
	}
}

func (pl *PeerList) DelPeer(endpoint netip.Prefix) {
	peer, ok := pl.peers[endpoint]
	if ok {
		peer.flags |= PifDelPending
	}
}

func (pl *PeerList) HasPeer(endpoint netip.Prefix) bool {
	peer, ok := pl.peers[endpoint]

	// Ignore not applied (disabled) and nodes already marked for deletion
	return ok && (peer.flags&PifDelPending|PifDisabled) == 0
}

func (pl *PeerList) GetPeer(endpoint netip.Prefix) (*PeerInfo, bool) {
	peer, ok := pl.peers[endpoint]
	return peer, ok
}

func (pl *PeerList) Delete(addrs ...netip.Prefix) {
	for _, ip := range addrs {
		delete(pl.peers, ip)
	}
}

func (pl *PeerList) Peers() []string {
	rv := []string{}

	for ip := range pl.peers {
		rv = append(rv, ip.String())
	}
	return rv
}

func (pl *PeerList) Count() int {
	return len(pl.peers)
}

func (pl *PeerList) Iterate(callback func(ip netip.Prefix, entry *PeerInfo)) {
	for ip, e := range pl.peers {
		callback(ip, e)
	}
}
