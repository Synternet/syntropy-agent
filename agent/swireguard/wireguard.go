/**
swireguard package is wireguard on steroids
super-wireguard, smart-wireguar, Syntropy-wireguard
This package is a helper for agent to configure
(kernel or userspace) wireguard tunnels
It also collects peer status, monitores latency, and other releated work
**/
package swireguard

import (
	"strings"
	"sync"

	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/peermon"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const pkgName = "Wireguard. "

type Wireguard struct {
	sync.RWMutex
	wgc         *wgctrl.Client
	peerMonitor *peermon.PeerMonitor
	router      common.SdnRouter
	// NOTE: caching wireguard setup may sound like an overhead at first.
	// But in future we may need to add checking/syncing/recreating delete interfaces
	// TODO: thing about using sync.Map here and get rid of mutex
	devices []*InterfaceInfo
}

// TODO: review and redesign Wireguard implementation.
// Maybe it should be an object, containing WG interface data and separate objects per interface ?
func New(r common.SdnRouter, pm *peermon.PeerMonitor) (*Wireguard, error) {
	wgc, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	wg := Wireguard{
		wgc:         wgc,
		peerMonitor: pm,
		router:      r,
	}

	return &wg, nil
}

func (wg *Wireguard) PeersMonitor() multiping.PingClient {
	return wg.peerMonitor
}

//func (wg *Wireguard) Router() common.Router {
//	return wg.router
//}

func (wg *Wireguard) Devices() ([]*wgtypes.Device, error) {
	rv := []*wgtypes.Device{}
	devs, err := wg.wgc.Devices()
	if err != nil {
		return nil, err
	}
	for _, d := range devs {
		if strings.HasPrefix(d.Name, env.InterfaceNamePrefix) {
			rv = append(rv, d)
		}
	}
	return rv, nil
}

func (wg *Wireguard) Close() error {
	return wg.wgc.Close()
}
