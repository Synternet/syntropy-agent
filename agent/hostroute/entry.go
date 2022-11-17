package hostroute

type routeEntry struct {
	count   uint32
	pending bool
}

func newEntry() *routeEntry {
	return &routeEntry{
		count:   0,    // count of duplicate route entries
		pending: true, // route is wanted to be added, but not yet done
	}
}
