package mole

type peerIDs struct {
	groupID      int
	connectionID int
}

type storage struct {
	peers  map[string]peerIDs
	ifaces map[string]string
}

func (s *storage) init() {
	s.peers = make(map[string]peerIDs)
	s.ifaces = make(map[string]string)
}
