package wireguard

import (
	"fmt"
	"io/ioutil"

	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// TODO this helper function should be removed in future
func (wg *Wireguard) getPrivateKey(ifname string) (key wgtypes.Key, err error) {
	privateFileName := config.AgentConfigDir + "/privatekey-" + ifname

	// at first try to read cached key
	strKey, err := ioutil.ReadFile(privateFileName)
	if err == nil {
		key, err = wgtypes.ParseKey(string(strKey))
		if err != nil {
			// Could not parse key. Most probably cache file is corrupted.
			// Do not exit and create a new key
			// (continue to new key generation fallback)
			logger.Warning().Println(pkgName, "cached key error: ", err)
		}
	}

	// Fallback to new key generation
	if err != nil {
		key, err = wgtypes.GeneratePrivateKey()
		if err != nil {
			return key, fmt.Errorf("generate private key error: %s", err.Error())
		}

		// cache for future reuse
		err = ioutil.WriteFile(privateFileName, []byte(key.String()), 0600)
		if err != nil {
			logger.Debug().Println(pkgName, "Caching private key error: ", err)
		}
	}

	return key, nil
}

func (wg *Wireguard) deviceUnlocked(ifname string) *InterfaceInfo {
	for _, dev := range wg.devices {
		if dev.IfName == ifname {
			// Additional check if OS configuration matches agent setup
			// (a kind of monitoring if interface is still present in OS)
			_, err := wg.wgc.Device(ifname)
			if err != nil {
				logger.Error().Println(pkgName, "expected interface does not exist", ifname)
			}
			return dev
		}
	}
	return nil
}

func (wg *Wireguard) interfaceCacheAdd(ii *InterfaceInfo) {
	if dev := wg.Device(ii.IfName); dev != nil {
		// Do not add another existing interface
		// TODO: think and add updating Keys, IP and Port
		return
	}
	wg.Lock()
	wg.devices = append(wg.devices, ii)
	wg.Unlock()
}

func (wg *Wireguard) interfaceCacheDel(ii *InterfaceInfo) {
	wg.Lock()
	defer wg.Unlock()

	deldev := func(index int) {
		wg.devices[index] = wg.devices[len(wg.devices)-1]
		wg.devices = wg.devices[:len(wg.devices)-1]

	}

	for idx, dev := range wg.devices {
		if dev.IfName == ii.IfName {
			deldev(idx)
			return
		}
		// TODO: maybe add elseif and check by private/public key ?
	}
}

func (wg *Wireguard) peerCacheAdd(pi *PeerInfo) {
	wg.Lock()
	defer wg.Unlock()

	dev := wg.deviceUnlocked(pi.IfName)
	if dev == nil {
		// Cannot add peer to non-existing interface
		// I don't need error here. This case should be handled before and should never reach here
		return
	}

	for _, p := range dev.peers {
		// PublicKey should be unique per peer
		if p.PublicKey == pi.PublicKey {
			return
		}
	}
	dev.peers = append(dev.peers, pi)
}

func (wg *Wireguard) peerCacheDel(pi *PeerInfo) {
	wg.Lock()
	defer wg.Unlock()

	dev := wg.deviceUnlocked(pi.IfName)
	if dev == nil {
		// Cannot remove peer from non-existing interface
		// I don't need error here. This case should be handled before and should never reach here
		return
	}

	for idx, p := range dev.peers {
		// PublicKey should be unique per peer
		if p.PublicKey == pi.PublicKey {
			dev.peers[idx] = dev.peers[len(dev.peers)-1]
			dev.peers = dev.peers[:len(dev.peers)-1]
		}
	}
}
