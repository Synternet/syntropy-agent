package routeselector

import "net/netip"

type SelectedRoute struct {
	IP     netip.Addr // best route IP address
	ID     int        // ConnectionID of the best route
	Reason *RouteChangeReason
}

type PathSelector interface {
	BestPath() *SelectedRoute
}
