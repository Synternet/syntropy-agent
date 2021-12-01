package servicemon

type routeFlag uint8

const (
	rfNone       = routeFlag(0x00)
	rfPendingAdd = routeFlag(0x01)
	rfPendingDel = routeFlag(0x02)
)

// The route entry. Destination will be map key
type routeEntry struct {
	ifname       string
	gateway      string
	connectionID int
	groupID      int
	flags        routeFlag
}

func (re *routeEntry) SetFlag(f routeFlag) {
	re.flags = re.flags | f
}

func (re *routeEntry) CheckFlag(f routeFlag) bool {
	return (re.flags & f) == f
}

func (re *routeEntry) ClearFlags() {
	re.flags = rfNone
}
