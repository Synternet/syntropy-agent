package servicemon

import (
	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

func (sm *ServiceMonitor) Reroute(newgw string) []*peeradata.Entry {
	peersActiveData := []*peeradata.Entry{}

	sm.Lock()
	defer sm.Unlock()

	for dest, routes := range sm.routes {
		if routes.Count() <= 1 {
			// cannot do smart routing on only one route list
			continue
		}

		ret := routes.Reroute(newgw, dest)
		if ret != nil {
			peersActiveData = append(peersActiveData, ret)
		}
	}

	return peersActiveData
}

// Reroute one routeList (aka Service Group)
func (rl *routeList) Reroute(newGw, destination string) *peeradata.Entry {
	newRoute := rl.Find(newGw)
	activeRoute := rl.GetActive()

	if newRoute == nil || activeRoute == nil {
		logger.Error().Println(pkgName, "No new or old active route is present.")
		return nil
	}
	if newRoute == activeRoute {
		return nil
	}

	// Change the route to new active
	err := netcfg.RouteReplace(newRoute.ifname, "", destination)
	if err != nil {
		logger.Error().Println(pkgName, "could not change routes to", destination, "via", newGw)
	}
	// reset active flags
	newRoute.SetFlag(rfActive)
	activeRoute.ClearFlags(rfActive)

	// Return route change
	return peeradata.NewEntry(activeRoute.connectionID, newRoute.connectionID, newRoute.groupID)
}
