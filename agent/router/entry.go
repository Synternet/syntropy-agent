package router

type routeEntry struct {
	gw    string
	iface string
}

type routeList struct {
	list   []*routeEntry
	active int
}

func (rl *routeList) Count() int {
	return len(rl.list)
}

func (rl *routeList) Add(re *routeEntry) {
	rl.list = append(rl.list, re)
}

func (rl *routeList) Del(idx int) {
	if idx >= len(rl.list) {
		return
	}

	rl.list[idx] = rl.list[len(rl.list)-1]
	rl.list = rl.list[:len(rl.list)-1]
}
