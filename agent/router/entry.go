package router

type bits uint8

const (
	agent  bits = 1 << iota // agent flag markes route added by this app
	active                  // active show that this route is curretly enabled
)

func (flags *bits) set(b bits) {
	*flags |= b
}

func (flags *bits) clear(b bits) {
	*flags &= ^b
}

func (flags *bits) has(b bits) bool {
	return *flags&b == b
}

type routeEntry struct {
	gw    string
	iface string
	flags bits // TODO: Do I really need these flags ??
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
