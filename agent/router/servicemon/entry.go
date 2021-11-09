package servicemon

import "github.com/SyntropyNet/syntropy-agent-go/internal/logger"

// The route entry. Destination will be map key
type routeEntry struct {
	ifname       string
	gateway      string
	connectionID int
	groupID      int
}

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
func (rl *routeList) Print() {
	for i, r := range rl.list {
		mark := " "
		if i == rl.active {
			mark = "*"
		}
		logger.Debug().Printf("%s [%d] %s %s (%d / %d)\n",
			mark, i, r.gateway, r.ifname, r.connectionID, r.groupID)
	}
}

func (rl *routeList) Count() int {
	return len(rl.list)
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

	rl.list = append(rl.list, re)
}

func (rl *routeList) Del(idx int) {
	if idx >= len(rl.list) {
		return
	}

	rl.list[idx] = rl.list[len(rl.list)-1]
	rl.list = rl.list[:len(rl.list)-1]
}
