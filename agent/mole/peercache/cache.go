package peercache

import (
	"fmt"
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
)

type peerCacheEntry struct {
	groupID      int
	connectionID int
	address      netip.Addr
}

// Controller sends full info only when adding peers
// When deleting peers there is only a public key in configuration json
// And I need this cache to get a connectionID, connection group ID (GID)
// And public IP address to delete created host route
type PeerCache struct {
	// No locking here. Caller is responsible for locks
	peers map[string]peerCacheEntry
}

func New() *PeerCache {
	pc := &PeerCache{
		peers: make(map[string]peerCacheEntry),
	}
	return pc
}

func (pc *PeerCache) Add(pi *swireguard.PeerInfo) error {
	pc.peers[makeKey(pi)] = peerCacheEntry{
		groupID:      pi.GroupID,
		connectionID: pi.ConnectionID,
		address:      pi.IP,
	}

	return nil
}

// Check for peer in cache and get connection ID and GID from there
func (pc *PeerCache) CheckAndDel(pi *swireguard.PeerInfo) error {
	entry, ok := pc.peers[makeKey(pi)]

	if !ok {
		return fmt.Errorf("peer %s not found on %s", pi.PublicKey, pi.IfName)
	}

	pi.ConnectionID = entry.connectionID
	pi.GroupID = entry.groupID
	pi.IP = entry.address

	delete(pc.peers, makeKey(pi))

	return nil
}

func (pc *PeerCache) Flush() error {
	pc.peers = make(map[string]peerCacheEntry)
	return nil
}

func (pc *PeerCache) Close() error {
	return pc.Flush()
}

func makeKey(pi *swireguard.PeerInfo) string {
	return pi.IfName + pi.PublicKey
}
