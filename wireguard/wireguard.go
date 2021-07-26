package wireguard

import (
	"golang.zx2c4.com/wireguard/wgctrl"
)

// TODO: I'm trying to embed anonymous entry in my wireguard implementation/wrapper
// Hope I will get a good mic of stock wgctl and my extentions.
type Wireguard struct {
	*wgctrl.Client
}

func New() (*Wireguard, error) {
	wgc, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	wg := Wireguard{wgc}

	return &wg, nil
}
