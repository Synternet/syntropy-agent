package servicemon

import (
	"fmt"

	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

func (sm *ServiceMonitor) Apply() ([]*routestatus.Connection, []*peeradata.Entry) {
	var routeStatusCons []*routestatus.Connection
	var peersActiveData []*peeradata.Entry
	var routeStatus *routestatus.Connection
	var padEntry *peeradata.Entry
	var deleteIPs []string
	sm.Lock()
	defer sm.Unlock()

	for ip, rl := range sm.routes {
		add, del := rl.Pending()
		if add == 0 && del == 0 {
			// nothing to do for this group
			continue
		}

		count := rl.Count()
		logger.Info().Printf("%s Apply: add:%d, del:%d, count:%d\n",
			pkgName, add, del, count)
		rl.Dump()

		if add == count && del == 0 {
			routeStatus, padEntry = rl.SetRoute(ip)
		} else if del == count && add == 0 {
			routeStatus, padEntry = rl.ClearRoute(ip)
			// It is dangerous to delete map entry while iterating.
			// Put a mark for later deletion
			deleteIPs = append(deleteIPs, ip)
		} else {
			routeStatus, padEntry = rl.MergeRoutes(ip, sm.reroutePath.BestPath())
		}
		if routeStatus != nil {
			routeStatusCons = append(routeStatusCons, routeStatus)
		}
		if padEntry != nil {
			peersActiveData = append(peersActiveData, padEntry)
		}
	}

	// Safely remove deleted entries
	for _, ip := range deleteIPs {
		delete(sm.routes, ip)
	}

	return routeStatusCons, peersActiveData
}

func (rl *routeList) SetRoute(destination string) (*routestatus.Connection, *peeradata.Entry) {
	defer rl.resetPending()

	routeConflict, conflictIfName := netcfg.RouteConflict(destination)
	logger.Debug().Println(pkgName, "Apply/SetRoute ", destination)

	if !routeConflict {
		// clean case - no route conflict. Simply add the route
		route := rl.GetDefault()
		if route == nil {
			logger.Error().Println(pkgName, "No new route found to", destination)
			return nil, nil
		}
		// mark route as active
		route.SetFlag(rfActive)
		logger.Info().Println(pkgName, "Route add ", destination, " via ", route.gateway, "/", route.ifname)
		err := netcfg.RouteAdd(route.ifname, "", destination)
		routeRes := routestatus.NewEntry(destination, err)

		if err != nil {
			logger.Error().Println(pkgName, "route add error:", err)
		}
		return routestatus.NewConnection(route.connectionID, route.groupID, routeRes),
			peeradata.NewEntry(0, route.connectionID, route.groupID)
	}

	// Route already exists. Check if it was configured earlier and is valid
	for _, route := range rl.list {
		if route.ifname == conflictIfName {
			// Mark active route to keep cache in sync
			active := rl.GetActive()
			if active != nil {
				active.ClearFlags(rfActive)
			}
			route.SetFlag(rfActive)
			// Return route added OK
			return routestatus.NewConnection(route.connectionID, route.groupID,
					routestatus.NewEntry(destination, nil)),
				peeradata.NewEntry(0, route.connectionID, route.groupID)
		}
	}

	// Route exists but is unknown - inform error
	err := fmt.Errorf("route to %s exists on %s", destination, conflictIfName)
	logger.Error().Println(pkgName, "route add error:", err)
	return nil, nil
}

func (rl *routeList) ClearRoute(destination string) (*routestatus.Connection, *peeradata.Entry) {
	defer rl.resetPending()

	logger.Debug().Println(pkgName, "Apply/ClearRoute ", destination)
	route := rl.GetActive()

	err := netcfg.RouteDel(route.ifname, destination)
	if err != nil {
		logger.Error().Println(pkgName, destination, "route delete error", err)
	}

	return nil,
		peeradata.NewEntry(route.connectionID, 0, route.groupID)
}

func (rl *routeList) MergeRoutes(destination string, newgw string) (*routestatus.Connection, *peeradata.Entry) {
	defer rl.resetPending()

	logger.Debug().Println(pkgName, "Apply/MergeRoute ", destination)

	newRoute := rl.Find(newgw)
	activeRoute := rl.GetActive()
	if activeRoute == nil {
		// Should never happen. But print to log just in case.
		logger.Error().Println(pkgName, "No active route was present.")
	}

	// Build new list of new and old, but not deleted entries
	newList := []*routeEntry{}
	for _, e := range rl.list {
		if !e.CheckFlag(rfPendingDel) {
			newList = append(newList, e)
		}
	}
	// drop old list and keep updated list.
	defer func() { rl.list = newList }()

	// check if route change is needed
	if newRoute == nil || newRoute.CheckFlag(rfPendingDel) {
		logger.Error().Println(pkgName, "New active route marked for deletion.")
		// No new route - try fallback to default route
		newRoute = rl.GetDefault()
	}

	if activeRoute == newRoute {
		// nothing changed, nothing to inform.
		return nil, nil
	}

	// Should never happen. Actually this case should be handled in ServiceMonitor.Apply
	if newRoute == nil {
		// But print to log just in case.
		logger.Error().Println(pkgName, "No active new route was found.")
		return rl.ClearRoute(destination)
	}
	// Should never happen. Actually this case should be handled in ServiceMonitor.Apply
	if activeRoute == nil {
		// But print to log just in case.
		logger.Error().Println(pkgName, "No active old route was found.")
		return rl.SetRoute(destination)
	}

	// Reuse reroute function to do actual job
	return nil, rl.Reroute(newRoute.gateway, destination)
}
