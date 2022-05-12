package common

import (
	"fmt"
	"net/netip"
)

type SdnNetworkPath struct {
	Ifname       string     // Interface, on which setup route
	PublicKey    string     // Wireguard public key
	Gateway      netip.Addr // Gateway, via which access destination
	ConnectionID int        // Unique connection ID
	GroupID      int        // Route SDN group ID
}

func (sr *SdnNetworkPath) String() string {
	return fmt.Sprintf(" via %s on %s [%d:%d]", sr.Gateway.String(), sr.Ifname,
		sr.ConnectionID, sr.GroupID)
}
