package servicemon

import "fmt"

type routeFlag uint8

const (
	rfNone       = routeFlag(0x00)
	rfPendingAdd = routeFlag(0x01)
	rfPendingDel = routeFlag(0x02)
	rfActive     = routeFlag(0x80)
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

func (re *routeEntry) ClearFlags(flagMask routeFlag) {
	re.flags = re.flags & ^flagMask
}

func (re *routeEntry) String() string {
	flags := [3]byte{' ', ' ', ' '}
	if re.CheckFlag(rfActive) {
		flags[0] = '*'
	}
	if re.CheckFlag(rfPendingAdd) {
		flags[1] = '+'
	}
	if re.CheckFlag(rfPendingDel) {
		flags[2] = '-'
	}

	return fmt.Sprintf("[%s] %s %s (%d / %d)",
		flags, re.gateway, re.ifname, re.connectionID, re.groupID)
}
