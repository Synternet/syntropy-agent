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

	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"golang.zx2c4.com/wireguard/wgctrl"
)

const pkgName = "Wireguard. "

type Wireguard struct {
	// If true - remove resident non-syntropy created tunnels
	RemoveNonSyntropyInterfaces bool

	sync.RWMutex
	wgc *wgctrl.Client
	// NOTE: caching wireguard setup may sound like an overhead at first.
	// But in future we may need to add checking/syncing/recreating delete interfaces
	// TODO: thing about using sync.Map here and get rid of mutex
	devices []*InterfaceInfo
}

// New creates new instance of Wireguard configurer and monitor
func New() (*Wireguard, error) {
	wgc, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	wg := Wireguard{
		wgc:                         wgc,
		RemoveNonSyntropyInterfaces: false,
	}

	loadKernelModule()

	return &wg, nil
}

func (wg *Wireguard) Devices() []*InterfaceInfo {
	wg.RLock()
	defer wg.RUnlock()

	rv := []*InterfaceInfo{}

	rv = append(rv, wg.devices...)

	return rv
}

func (wg *Wireguard) Close() error {
	// If configured - cleanup created interfaces on exit.
	if config.CleanupOnExit() {
		for _, dev := range wg.devices {
			wg.RemoveInterface(dev)
		}
	}

	return wg.wgc.Close()
}

// Flush clears all WG local cache
func (wg *Wireguard) Flush() {
	wg.Lock()
	defer wg.Unlock()

	wg.devices = wg.devices[:0]
}

// Apply function setups cached WG configuration,
// and cleans up resident configuration
func (wg *Wireguard) Apply() error {
	wg.RLock()
	defer wg.RUnlock()

	osDevs, err := wg.wgc.Devices()
	if err != nil {
		return err
	}

	// remove resident devices (created by already terminated agent)
	for _, osDev := range osDevs {
		found := false
		for _, agentDev := range wg.devices {
			if osDev.Name == agentDev.IfName {
				found = true
				break
			}
		}

		if !found {
			if strings.HasPrefix(osDev.Name, env.InterfaceNamePrefix) ||
				wg.RemoveNonSyntropyInterfaces {
				wg.RemoveInterface(&InterfaceInfo{
					IfName: osDev.Name,
				})
			}
		}
	}

	// reread OS setup - it may has changed
	osDevs, err = wg.wgc.Devices()
	if err != nil {
		return err
	}
	// create missing devices
	for _, agentDev := range wg.devices {
		found := false
		for _, osDev := range osDevs {
			if osDev.Name == agentDev.IfName {
				found = true
			}
		}

		if !found {
			wg.CreateInterface(agentDev)
		}
		wg.applyPeers(agentDev)
	}

	return nil
}
