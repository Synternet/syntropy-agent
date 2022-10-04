package servicemon

import (
	"fmt"
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

func (sm *ServiceMonitor) Apply() ([]*routestatus.Connection, []*peeradata.Entry) {
	var routeStatusCons []*routestatus.Connection
	var peersActiveData []*peeradata.Entry
	var deleteIPs []netip.Prefix

	sm.Lock()
	defer sm.Unlock()

	bestRoute := sm.routeMonitor.BestPath()

	for ip, rl := range sm.routes {
		if rl.Disabled() {
			logger.Warning().Println(pkgName, "Apply Service ignores conflicting IP:", ip)
			continue
		}

		add, del := rl.Pending()
		if add == 0 && del == 0 {
			// nothing to do for this group
			continue
		}
		count := rl.Count()
		logger.Debug().Println(pkgName, "Apply Service", ip, ":", rl.String(), ".")

		if add == count && del == 0 {
			routeStatus, _ := rl.setRoute(ip)
			if routeStatus != nil {
				routeStatusCons = append(routeStatusCons, routeStatus)
			}
		} else if del == count && add == 0 {
			rl.clearRoute(ip)
			// It is dangerous to delete map entry while iterating.
			// Put a mark for later deletion
			deleteIPs = append(deleteIPs, ip)
		} else {
			// If route is valid - apply it.
			// Invalid IP means delete current route
			if bestRoute.IP.IsValid() {
				rl.mergeRoutes(ip, &bestRoute.IP)
			} else {
				rl.mergeRoutes(ip, nil)
			}
		}

	}

	// Safely remove deleted entries
	for _, ip := range deleteIPs {
		delete(sm.routes, ip)
	}

	// Format response message
	// Note: always send it, even if no services are configured
	newConnID := 0
	if bestRoute != nil {
		newConnID = bestRoute.ID
	}
	if sm.activeConnectionID != newConnID {
		peersActiveData = append(peersActiveData,
			peeradata.NewEntry(sm.activeConnectionID, newConnID, sm.groupID))
		sm.activeConnectionID = newConnID
	}

	return routeStatusCons, peersActiveData
}

func (sm *ServiceMonitor) ResolveIpConflict(isIPconflict func(netip.Prefix, int) bool) (count int) {
	sm.Lock()
	defer sm.Unlock()

	for ip, rl := range sm.routes {
		if rl.Disabled() {
			// check if IP conflict still present
			if !isIPconflict(ip, 0) { // TODO
				// clear disabled flag and increment updated services count
				rl.flags &= ^rlfDisabled
				count++
			}
		}
	}
	return count
}

func (rl *routeList) setRoute(destination netip.Prefix) (*routestatus.Connection, error) {
	defer rl.resetPending()

	routeConflict, conflictIfName := netcfg.RouteSearch(&destination)
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
		logger.Debug().Println(pkgName, "Route add ", destination, " via ", route.gateway, "/", route.ifname)
		err := netcfg.RouteAdd(route.ifname, nil, &destination)
		routeRes := routestatus.NewEntry(destination, err)

		if err != nil {
			logger.Error().Println(pkgName, "route add error:", err)
		}
		return routestatus.NewConnection(route.connectionID, rl.GroupID, routeRes),
			nil
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
			return routestatus.NewConnection(route.connectionID, rl.GroupID,
					routestatus.NewEntry(destination, nil)),
				nil
		}
	}

	// Route exists but is unknown - inform error
	err := fmt.Errorf("route to %s exists on %s", destination, conflictIfName)
	logger.Error().Println(pkgName, "route add error:", err)
	return nil, err
}

func (rl *routeList) clearRoute(destination netip.Prefix) error {
	defer rl.resetPending()

	logger.Debug().Println(pkgName, "Apply/ClearRoute", destination)

	route := rl.GetActive()
	if route == nil {
		return nil
	}

	err := netcfg.RouteDel(route.ifname, &destination)
	if err != nil {
		logger.Error().Println(pkgName, destination, "route delete error", err)
	}
	route.ClearFlags(rfActive)

	return nil
}

func (rl *routeList) mergeRoutes(destination netip.Prefix, newgw *netip.Addr) error {
	logger.Debug().Println(pkgName, "Apply/MergeRoute ", destination)

	activeRoute := rl.GetActive()
	var newRoute *routeEntry
	if newgw != nil && newgw.IsValid() {
		newRoute = rl.Find(*newgw)
		// check if route change is needed
		// I think this both cases should never happen
		if newRoute == nil {
			logger.Error().Println(pkgName, "New route ", newgw, "not found.")
		} else if newRoute.CheckFlag(rfPendingDel) {
			logger.Error().Println(pkgName, "New active route marked for deletion.", newgw)
			newRoute = nil
		}
	}

	// Build new list of new and old, but not deleted entries
	newList := []*routeEntry{}
	for _, e := range rl.list {
		if !e.CheckFlag(rfPendingDel) {
			newList = append(newList, e)
		}
	}
	// drop old list and keep updated list.
	rl.list = newList
	rl.resetPending()

	// Reuse reroute function to do actual job
	return rl.Reroute(newRoute, activeRoute, destination)
}
