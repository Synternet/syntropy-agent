package wireguard

import (
	"golang.zx2c4.com/wireguard/wgctrl"
)

// TODO: I'm trying to embed anonymous entry in my wireguard implementation/wrapper
// Hope I will get a good mic of stock wgctl and my extentions.
type Wireguard struct {
	*wgctrl.Client
}

// TODO: review and redesign Wireguard implementation.
// Maybe it should be an object, containing WG interface data and separate objects per interface ?
func New() (*Wireguard, error) {
	wgc, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	wg := Wireguard{wgc}

	return &wg, nil
}
