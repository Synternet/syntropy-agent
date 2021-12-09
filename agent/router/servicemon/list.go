package servicemon

import (
	"fmt"

	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

// Group or routes. Destination will be map key
type routeList struct {
	list   []*routeEntry
	active int
}

func newRouteList() *routeList {
	return &routeList{
		// when adding new destination - always start with the first route active
		// Yes, I know this is zero by default, but I wanted it to be explicitely clear
		active: 0,
	}
}
func (rl *routeList) Dump() {
	for i, r := range rl.list {
		mark := " "
		if i == rl.active {
			mark = "*"
		}
		logger.Debug().Printf("%s%s [%d] %s %s (%d / %d)\n",
			pkgName, mark, i, r.gateway, r.ifname, r.connectionID, r.groupID)
	}
}

// Returns total count of entries in this service route list
func (rl *routeList) Count() int {
	return len(rl.list)
}

// Returns pending entries to be added and/or deleted
func (rl *routeList) Pending() (add, del int) {
	for _, r := range rl.list {
		if r.CheckFlag(rfPendingAdd) {
			add++
		}
		if r.CheckFlag(rfPendingDel) {
			del++
		}
	}
	return add, del
}

func (rl *routeList) clearFlags() {
	for _, r := range rl.list {
		r.ClearFlags()
	}
}

// Searches for Public link.
// If not found - returns first in list
func (rl *routeList) GetDefault() *routeEntry {
	// TODO implement Public search.
	// Shorcut now - choose active link
	return rl.GetActive()
}

// Returns active route from the set
func (rl *routeList) GetActive() *routeEntry {
	if rl.Count() == 0 {
		return nil
	}
	return rl.list[rl.active]
}

func (rl *routeList) Add(re *routeEntry) {
	// Dupplicate entries happen when WSS connection was lost
	// and after reconnecting controller sends whole config
	for _, r := range rl.list {
		if r.gateway == re.gateway {
			// skip dupplicate entry
			return
		}
	}

	re.SetFlag(rfPendingAdd)
	rl.list = append(rl.list, re)
}

func (rl *routeList) MarkDel(gateway string) {
	// Dupplicate entries happen when WSS connection was lost
	// and after reconnecting controller sends whole config
	for _, r := range rl.list {
		if r.gateway == gateway {
			r.SetFlag(rfPendingDel)
			return
		}
	}
}

func (rl *routeList) Del(idx int) {
	if idx >= len(rl.list) {
		return
	}

	rl.list[idx] = rl.list[len(rl.list)-1]
	rl.list = rl.list[:len(rl.list)-1]
}

func (rl *routeList) SetRoute(destination string) (*routestatus.Connection, *peeradata.Entry) {
	defer rl.clearFlags()

	routeConflict, conflictIfName := netcfg.RouteConflict(destination)
	logger.Debug().Println(pkgName, "Apply/SetRoute ", destination)

	if !routeConflict {
		// clean case - no route conflict. Simply add the route
		route := rl.GetDefault()
		if route == nil {
			return nil, nil
		}
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
	for idx, route := range rl.list {
		if route.ifname == conflictIfName {
			// Mark active route to keep cache in sync
			rl.active = idx
			// Return route added OK
			return routestatus.NewConnection(route.connectionID, route.groupID,
					routestatus.NewEntry(destination, nil)),
				peeradata.NewEntry(0, route.connectionID, route.groupID)
		}
	}

	// Route exists but is unknown - inform error
	err := fmt.Errorf("route to %s exists on %s", destination, conflictIfName)
	logger.Error().Println(pkgName, "route add error:", err)
	route := rl.GetDefault()
	return routestatus.NewConnection(route.connectionID, route.groupID,
			routestatus.NewEntry(destination, err)),
		nil
}

func (rl *routeList) ClearRoute(destination string) (*routestatus.Connection, *peeradata.Entry) {
	defer rl.clearFlags()

	logger.Debug().Println(pkgName, "Apply/ClearRoute ", destination)
	route := rl.list[rl.active]

	err := netcfg.RouteDel(route.ifname, destination)
	if err != nil {
		logger.Error().Println(pkgName, destination, "route delete error", err)
	}

	return nil,
		peeradata.NewEntry(route.connectionID, 0, route.groupID)
}

func (rl *routeList) MergeRoutes(destination string) (*routestatus.Connection, *peeradata.Entry) {
	defer rl.clearFlags()

	logger.Debug().Println(pkgName, "Apply/MergeRoute ", destination, "Not Implemented Yet !")
	// TODO implement me
	return nil, nil
}
