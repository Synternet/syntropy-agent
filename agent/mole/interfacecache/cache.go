package interfacecache

import (
	"fmt"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"net/netip"
	"strconv"
)

type interfaceCacheEntry struct {
	Ifname  string
	Address netip.Addr
}

// Controller sends full info only when adding peers
// When deleting peers there is only a public key in configuration json
// And I need this cache to get a publickey
// And public IP address to delete created host route
type InterfaceCache struct {
	// No locking here. Caller is responsible for locks
	interfaces map[int]interfaceCacheEntry
}

func New() *InterfaceCache {
	ic := &InterfaceCache{
		interfaces: make(map[int]interfaceCacheEntry),
	}
	return ic
}

func (ic *InterfaceCache) Add(ii *swireguard.InterfaceInfo) error {
	ic.interfaces[ii.IfIndex] = interfaceCacheEntry{
		Ifname:  ii.IfName,
		Address: ii.IP,
	}

	return nil
}

// return interface entry by index
func (ic *InterfaceCache) GetInterfaceByIndex(index int) (*interfaceCacheEntry, error) {
	entry, ok := ic.interfaces[index]
	if !ok {
		return nil, fmt.Errorf("interfce not found for index: %s", strconv.Itoa(index))
	}

	return &entry, nil
}

func (ic *InterfaceCache) Flush() error {
	ic.interfaces = make(map[int]interfaceCacheEntry)
	return nil
}

func (ic *InterfaceCache) Close() error {
	return ic.Flush()
}
