package servicemon

import (
	"fmt"
	"sync"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
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
