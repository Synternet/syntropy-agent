package common

import "fmt"

type SdnNetworkPath struct {
	// Interface, on which setup route
	Ifname string
	// Gateway, via which access destination
	Gateway string
	// connection/tunnel ID
	ID int
}

func (sr *SdnNetworkPath) String() string {
	return fmt.Sprintf(" via %s on %s [%d]", sr.Gateway, sr.Ifname, sr.ID)
}

type SdnRouter interface {
	RouteAdd(route *SdnNetworkPath, dest ...string) error
	RouteDel(route *SdnNetworkPath, dest ...string) error
}
