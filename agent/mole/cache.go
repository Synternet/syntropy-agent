package mole

import "net/netip"

type peerCacheEntry struct {
	groupID      int
	connectionID int
	destIP       netip.Prefix
	gateway      string
	gwIfname     string
}

type storage struct {
	controller []peerCacheEntry
	peers      map[string]peerCacheEntry
	ifaces     map[string]string
}

func (s *storage) init() {
	s.peers = make(map[string]peerCacheEntry)
	s.ifaces = make(map[string]string)
	s.controller = nil
}
