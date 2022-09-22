package servicemon

import (
	"fmt"
	"net/netip"
	"sync"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const pkgName = "ServiceMonitor. "

type ServiceMonitor struct {
	sync.Mutex
	routes             map[netip.Prefix]*routeList
	routeMonitor       peermon.PathSelector
	groupID            int
	activeConnectionID int
}

func New(ps peermon.PathSelector, gid int) *ServiceMonitor {
	return &ServiceMonitor{
		routes:             make(map[netip.Prefix]*routeList),
		routeMonitor:       ps,
		groupID:            gid,
		activeConnectionID: 0,
	}
}

func (sm *ServiceMonitor) Add(netpath *common.SdnNetworkPath, ip netip.Prefix, disabled bool) error {
	sm.Lock()
	defer sm.Unlock()

	// Keep a list of active SDN routes
	if sm.routes[ip] == nil {
		sm.routes[ip] = newRouteList(netpath.GroupID, disabled)
	}

	sm.routes[ip].Add(&routeEntry{
		ifname:       netpath.Ifname,
		publicKey:    netpath.PublicKey,
		gateway:      netpath.Gateway,
		connectionID: netpath.ConnectionID,
	})

	return nil
}

func (sm *ServiceMonitor) Del(netpath *common.SdnNetworkPath, ip netip.Prefix) error {
	sm.Lock()
	defer sm.Unlock()

	// Keep a list of active SDN routes
	if sm.routes[ip] == nil {
		return fmt.Errorf("no such address %s", ip)
	}

	sm.routes[ip].MarkDel(netpath.Gateway)

	return nil
}

func (sm *ServiceMonitor) HasAddress(ip netip.Prefix) bool {
	sm.Lock()
	defer sm.Unlock()

	rl, ok := sm.routes[ip]
	if ok && !rl.Disabled() {
		return true
	}

	return false
}

func (sm *ServiceMonitor) Close() error {
	sm.Lock()
	defer sm.Unlock()

	for ip, rl := range sm.routes {
		if rl.Disabled() {
			// no need to delete routes that were not added
			// conflicting IP was detected (and prevented)
			continue
		}

		rl.clearRoute(ip)
	}

	// delete map entries
	for ip := range sm.routes {
		delete(sm.routes, ip)
	}

	return nil
}

func (sm *ServiceMonitor) Flush() {
	sm.Lock()
	defer sm.Unlock()

	var deleteIPs []netip.Prefix

	for ip, rl := range sm.routes {
		if rl.Disabled() {
			// no need to do smart routes delete and merge for routes that were not added
			// because conflicting IP was detected (and prevented)
			// instead flush them asap
			deleteIPs = append(deleteIPs, ip)
			continue
		}

		logger.Debug().Println(pkgName, "Flushing", ip)
		rl.Flush()
	}

	// now flush/delete the conflicting addresses
	for _, ip := range deleteIPs {
		logger.Debug().Println(pkgName, "Flushing (previously IP conflict)", ip)
		delete(sm.routes, ip)
	}
}
