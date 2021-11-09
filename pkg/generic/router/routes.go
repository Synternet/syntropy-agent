// Generic router interface
package router

import "fmt"

type SdnNetworkPath struct {
	Ifname       string // Interface, on which setup route
	Gateway      string // Gateway, via which access destination
	ConnectionID int
	GroupID      int
}

func (sr *SdnNetworkPath) String() string {
	return fmt.Sprintf(" via %s on %s [%d : %d]", sr.Gateway, sr.Ifname,
		sr.ConnectionID, sr.GroupID)
}

type RouteResult struct {
	IP    string
	Error error
}

type SdnRouter interface {
	RouteAdd(route *SdnNetworkPath, dest []string) []RouteResult
	RouteDel(route *SdnNetworkPath, dest []string) []RouteResult
}
