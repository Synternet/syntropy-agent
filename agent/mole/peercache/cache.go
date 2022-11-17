package peercache

import (
	"fmt"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"net/netip"
	"strconv"
)

type peerCacheEntry struct {
	GroupID      int
	ConnectionID int
	IfName       string
	IfIndex      int
	PublicKey    string
	Gateway      netip.Addr
	Address      netip.Addr
	AllowedIPs   []netip.Prefix
}

// Controller sends full info only when adding peers
// When deleting peers there is only a public key in configuration json
// And I need this cache to get a publickey
// And public IP address to delete created host route
type PeerCache struct {
	// No locking here. Caller is responsible for locks
	peers map[int]peerCacheEntry
}

func New() *PeerCache {
	pc := &PeerCache{
		peers: make(map[int]peerCacheEntry),
	}
	return pc
}

func (pc *PeerCache) Add(pi *swireguard.PeerInfo) error {
	pc.peers[makeKey(pi)] = peerCacheEntry{
		GroupID:    pi.GroupID,
		PublicKey:  pi.PublicKey,
		Address:    pi.IP,
		Gateway:    pi.Gateway,
		IfName:     pi.IfName,
		IfIndex:    pi.IfIndex,
		AllowedIPs: pi.AllowedIPs,
	}

	return nil
}

// Check for peer in cache and get connection ID and GID from there
func (pc *PeerCache) CheckAndDel(pi *swireguard.PeerInfo) error {
	entry, ok := pc.peers[makeKey(pi)]

	if !ok {
		return fmt.Errorf("peer %s not found on %s", pi.PublicKey, pi.IfName)
	}

	pi.PublicKey = entry.PublicKey
	pi.IP = entry.Address
	pi.ConnectionID = entry.ConnectionID
	pi.GroupID = entry.GroupID

	delete(pc.peers, makeKey(pi))

	return nil
}

// Check if allowedIp does not already exist in cache and if not update it
func (pc *PeerCache) AddPeerAllowedIps(peerCache *peerCacheEntry, allowedIp netip.Prefix) {
	for _, aip := range peerCache.AllowedIPs {
		if aip == allowedIp {
			return
		}
		peerCache.AllowedIPs = append(peerCache.AllowedIPs, allowedIp)

	}
}

// Find and delete allowedIp from cache
func (pc *PeerCache) RemovePeerAllowedIps(peerCache *peerCacheEntry, allowedIp netip.Prefix) {
	for i, aip := range peerCache.AllowedIPs {
		if aip == allowedIp {
			peerCache.AllowedIPs = append(peerCache.AllowedIPs[:i], peerCache.AllowedIPs[i+1:]...)
		}
	}
}

// return peer entry by id
func (pc *PeerCache) GetPeerByConnectionID(connectionID int) (*peerCacheEntry, error) {
	entry, ok := pc.peers[connectionID]
	if !ok {
		return nil, fmt.Errorf("peer not found for connectionId: %s", strconv.Itoa(connectionID))
	}

	return &entry, nil
}

func (pc *PeerCache) GetPeerInfoByConnectionID(connectionID int) (*swireguard.PeerInfo, error) {
	entry, ok := pc.peers[connectionID]
	if !ok {
		return nil, fmt.Errorf("peer not found for connectionId: %s", strconv.Itoa(connectionID))
	}

	return &swireguard.PeerInfo{
		GroupID:    entry.GroupID,
		PublicKey:  entry.PublicKey,
		IP:         entry.Address,
		Gateway:    entry.Gateway,
		IfName:     entry.IfName,
		IfIndex:    entry.IfIndex,
		AllowedIPs: entry.AllowedIPs,
	}, nil
}

func (pc *PeerCache) Flush() error {
	pc.peers = make(map[int]peerCacheEntry)
	return nil
}

func (pc *PeerCache) Close() error {
	return pc.Flush()
}

func makeKey(pi *swireguard.PeerInfo) int {
	return pi.ConnectionID
}
