package common

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
