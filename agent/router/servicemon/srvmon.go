package servicemon

import (
	"fmt"
	"sync"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const pkgName = "ServiceMonitor. "

type PathSelector interface {
	BestPath() string
}

type ServiceMonitor struct {
	sync.Mutex
	routes      map[string]*routeList
	reroutePath PathSelector
}

func New(ps PathSelector) *ServiceMonitor {
	return &ServiceMonitor{
		routes:      make(map[string]*routeList),
		reroutePath: ps,
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
