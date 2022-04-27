package servicemon

import (
	"fmt"
	"sync"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const pkgName = "ServiceMonitor. "

type ServiceMonitor struct {
	sync.Mutex
	routes             map[string]*routeList
	routeMonitor       peermon.PathSelector
	groupID            int
	activeConnectionID int
}

func New(ps peermon.PathSelector, gid int) *ServiceMonitor {
	return &ServiceMonitor{
		routes:             make(map[string]*routeList),
		routeMonitor:       ps,
		groupID:            gid,
		activeConnectionID: 0,
	}
}

func (sm *ServiceMonitor) Add(netpath *common.SdnNetworkPath, ip string) error {
	sm.Lock()
	defer sm.Unlock()

	// Keep a list of active SDN routes
	if sm.routes[ip] == nil {
		sm.routes[ip] = newRouteList()
	}
	sm.routes[ip].Add(&routeEntry{
		ifname:       netpath.Ifname,
		publicKey:    netpath.PublicKey,
		gateway:      netpath.Gateway,
		connectionID: netpath.ConnectionID,
		groupID:      netpath.GroupID,
	})

	return nil
}

func (sm *ServiceMonitor) Del(netpath *common.SdnNetworkPath, ip string) error {
	sm.Lock()
	defer sm.Unlock()

	// Keep a list of active SDN routes
	if sm.routes[ip] == nil {
		return fmt.Errorf("no such address %s", ip)
	}

	sm.routes[ip].MarkDel(netpath.Gateway)

	return nil
}

func (sm *ServiceMonitor) Close() error {
	sm.Lock()
	defer sm.Unlock()

	for ip, rl := range sm.routes {
		rl.ClearRoute(ip)
	}
	return nil
}

func (sm *ServiceMonitor) Flush() {
	sm.Lock()
	defer sm.Unlock()

	for ip, rl := range sm.routes {
		logger.Debug().Println(pkgName, "Flushing", ip)
		rl.Flush()
	}
}
