package mole

import "net/netip"

type peerCacheEntry struct {
	groupID      int
	connectionID int
	destIP       netip.Prefix
	gateway      netip.Addr
	gwIfname     string
}

type storage struct {
	peers  map[string]peerCacheEntry
	ifaces map[string]netip.Addr
}

func (s *storage) init() {
	s.peers = make(map[string]peerCacheEntry)
	s.ifaces = make(map[string]netip.Addr)
}
