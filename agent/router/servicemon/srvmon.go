package servicemon

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
)

// Not a fatal error, but a message for higher layer (analog to EOF)
var ErrSdnRouteExists = errors.New("sdn route exists")

const pkgName = "ServiceMonitor. "

type ServiceMonitor struct {
	sync.Mutex
	writer io.Writer
	routes map[string]*routeList
}

func New(w io.Writer) *ServiceMonitor {
	return &ServiceMonitor{
		writer: w,
		routes: make(map[string]*routeList),
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

	if sm.routes[ip].Count() > 1 {
		return ErrSdnRouteExists
	}
	return nil
}

func (sm *ServiceMonitor) Del(netpath *common.SdnNetworkPath, ip string) error {
	sm.Lock()
	defer sm.Unlock()

	// Keep a list of active SDN routes
	if sm.routes[ip] == nil {
		return fmt.Errorf("no such address %s", ip)
	}

	for idx, entry := range sm.routes[ip].list {
		if entry.gateway == netpath.Gateway {
			sm.routes[ip].Del(idx)
		}
	}

	if sm.routes[ip].Count() == 0 {
		delete(sm.routes, ip)
		return nil
	}

	return nil
}
