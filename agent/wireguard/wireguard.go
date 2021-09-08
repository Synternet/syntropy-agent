// wireguard package is a helper for agent to configure
// (kernel or userspace) wireguard tunnels
package wireguard

import (
	"github.com/SyntropyNet/syntropy-agent-go/internal/sdn"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const pkgName = "Wireguard. "

// TODO: I'm trying to embed anonymous entry in my wireguard implementation/wrapper
// Hope I will get a good mic of stock wgctl and my extentions.
type Wireguard struct {
	wgc    *wgctrl.Client
	sdn    *sdn.SdnMonitor
	router common.SdnRouter
}

// TODO: review and redesign Wireguard implementation.
// Maybe it should be an object, containing WG interface data and separate objects per interface ?
func New(r common.SdnRouter, s *sdn.SdnMonitor) (*Wireguard, error) {
	wgc, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	wg := Wireguard{
		wgc:    wgc,
		sdn:    s,
		router: r,
	}

	return &wg, nil
}

//func (wg *Wireguard) Sdn() *sdn.SdnMonitor {
//	return wg.sdn
//}

//func (wg *Wireguard) Router() common.Router {
//	return wg.router
//}

func (wg *Wireguard) Devices() ([]*wgtypes.Device, error) {
	return wg.wgc.Devices()
}

func (wg *Wireguard) Close() error {
	return wg.wgc.Close()
}
