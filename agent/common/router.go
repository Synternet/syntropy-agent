package common

import "fmt"

type SdnNetworkPath struct {
	Ifname       string // Interface, on which setup route
	PublicKey    string // Wireguard public key
	Gateway      string // Gateway, via which access destination
	ConnectionID int    // Unique connection ID
	GroupID      int    // Route SDN group ID
}

func (sr *SdnNetworkPath) String() string {
	return fmt.Sprintf(" via %s on %s [%d : %d]", sr.Gateway, sr.Ifname,
		sr.ConnectionID, sr.GroupID)
}
