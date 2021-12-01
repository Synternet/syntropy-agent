package servicemon

import (
	"fmt"
	"sync"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent-go/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
)

const pkgName = "ServiceMonitor. "

type ServiceMonitor struct {
	sync.Mutex
	routes map[string]*routeList
}

func New() *ServiceMonitor {
	return &ServiceMonitor{
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

func (sm *ServiceMonitor) Apply() ([]*routestatus.Connection, []*peeradata.Entry) {
	var routeStatusCons []*routestatus.Connection
	var peersActiveData []*peeradata.Entry
	var routeStatus *routestatus.Connection
	var padEntry *peeradata.Entry
	sm.Lock()
	defer sm.Unlock()

	for ip, rl := range sm.routes {
		add, del := rl.Pending()
		count := rl.Count()
		logger.Info().Printf("%s Apply: add:%d, del:%d, count:%d\n", pkgName,
			add, del, count)
		if add == 0 && del == 0 {
			// nothing to do for this group
			continue
		} else if add == count && del == 0 {
			routeStatus, padEntry = rl.SetRoute(ip)
		} else if del == count && add == 0 {
			routeStatus, padEntry = rl.ClearRoute(ip)
		} else {
			routeStatus, padEntry = rl.MergeRoutes(ip)
		}
		if routeStatus != nil {
			routeStatusCons = append(routeStatusCons, routeStatus)
		}
		if padEntry != nil {
			peersActiveData = append(peersActiveData, padEntry)
		}
	}

	return routeStatusCons, peersActiveData
}
