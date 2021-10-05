/**
swireguard package is wireguard on steroids
super-wireguard, smart-wireguar, Syntropy-wireguard
This package is a helper for agent to configure
(kernel or userspace) wireguard tunnels
It also collects peer status, monitores latency, and other releated work
**/
package swireguard

import (
	"sync"

	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/internal/peermon"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
	"golang.zx2c4.com/wireguard/wgctrl"
)

const pkgName = "Wireguard. "

type Wireguard struct {
	sync.RWMutex
	wgc         *wgctrl.Client
	peerMonitor *peermon.PeerMonitor
	// NOTE: caching wireguard setup may sound like an overhead at first.
	// But in future we may need to add checking/syncing/recreating delete interfaces
	// TODO: thing about using sync.Map here and get rid of mutex
	devices []*InterfaceInfo
}

// TODO: review and redesign Wireguard implementation.
// Maybe it should be an object, containing WG interface data and separate objects per interface ?
func New(pm *peermon.PeerMonitor) (*Wireguard, error) {
	wgc, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	wg := Wireguard{
		wgc:         wgc,
		peerMonitor: pm,
	}

	return &wg, nil
}

func (wg *Wireguard) PeersMonitor() multiping.PingClient {
	return wg.peerMonitor
}

func (wg *Wireguard) Devices() []*InterfaceInfo {
	rv := []*InterfaceInfo{}

	rv = append(rv, wg.devices...)

	return rv
}

func (wg *Wireguard) Close() error {
	// If configured - cleanup created interfaces on exit.
	if config.CleanupOnExit() {
		logger.Info().Println(pkgName, "deleting wireguard tunnels.")
		for _, dev := range wg.devices {
			wg.RemoveInterface(dev)
		}
	} else {
		logger.Info().Println(pkgName, "keeping wireguard tunnels on exit.")
	}

	return wg.wgc.Close()
}
