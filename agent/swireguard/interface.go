package swireguard

import (
	"fmt"
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type InterfaceInfo struct {
	IfName     string
	PublicKey  string
	privateKey string
	IP         netip.Addr
	Port       int
	peers      []*PeerInfo
}

func (ii *InterfaceInfo) Peers() []*PeerInfo {
	rv := []*PeerInfo{}
	rv = append(rv, ii.peers...)
	return rv
}

// Remove all peers
func (ii *InterfaceInfo) flushPeers() {
	ii.peers = ii.peers[:0]
}

func (wg *Wireguard) Device(ifname string) *InterfaceInfo {
	wg.RLock()
	defer wg.RUnlock()
	return wg.deviceUnlocked(ifname)
}

func (wg *Wireguard) interfaceExist(ifname string) bool {
	return wg.Device(ifname) != nil
}

func (wg *Wireguard) CreateInterface(ii *InterfaceInfo) error {
	if ii == nil {
		return fmt.Errorf("invalid parameters to CreateInterface")
	}

	var err error
	var privKey wgtypes.Key
	var port int
	myDev := wg.Device(ii.IfName)
	osDev, _ := wg.wgc.Device(ii.IfName)

	if myDev == nil {
		// Alloc new cached device and add to cache
		myDev = &InterfaceInfo{
			IfName:    ii.IfName,
			PublicKey: ii.PublicKey,
			Port:      ii.Port,
			IP:        ii.IP,
		}
		wg.interfaceCacheAdd(myDev)
	} else {
		wg.Lock()
		myDev.flushPeers()
		wg.Unlock()
	}

	if osDev == nil {
		// create interface, if missing
		logger.Debug().Println(pkgName, "create interface", ii.IfName)
		err = wg.createInterface(ii.IfName)
		if err != nil {
			return fmt.Errorf("create wg interface failed: %s", err.Error())
		}
		privKey, err = wgtypes.GeneratePrivateKey()
		if err != nil {
			return fmt.Errorf("generate private key error: %s", err.Error())
		}
		port = findFreePort(ii.Port)
	} else {
		// reuse existing interface configuration
		logger.Debug().Println(pkgName, "reusing existing interface", ii.IfName)
		privKey = osDev.PrivateKey
		if isPortInRange(osDev.ListenPort) {
			port = osDev.ListenPort
		} else {
			port = findFreePort(ii.Port)
		}
	}

	wgconf := wgtypes.Config{
		PrivateKey: &privKey,
	}
	if port > 0 {
		wgconf.ListenPort = &port
	}

	err = wg.wgc.ConfigureDevice(ii.IfName, wgconf)
	if err != nil {
		return fmt.Errorf("configure interface failed: %s", err.Error())
	}

	// Reread OS configuration and update cache for params, that may have changed
	osDev, err = wg.wgc.Device(ii.IfName)
	if err != nil {
		return fmt.Errorf("reading wg device info error: %s", err.Error())
	}

	// Finally updata params, thay may have changed
	myDev.Port = osDev.ListenPort
	myDev.privateKey = osDev.PrivateKey.String()
	myDev.PublicKey = osDev.PublicKey.String()

	ii.Port = myDev.Port
	ii.PublicKey = myDev.PublicKey

	return nil
}

func (wg *Wireguard) RemoveInterface(ii *InterfaceInfo) error {
	if ii == nil {
		return fmt.Errorf("invalid parameters to RemoveInterface")
	}

	// Delete from cache
	dev := wg.Device(ii.IfName)
	if dev != nil {
		logger.Debug().Println(pkgName, "Remove interface", ii.IfName)
		wg.interfaceCacheDel(dev)
	} else {
		logger.Warning().Println(pkgName, "Remove interface", ii.IfName, "not found in cache.")
	}

	// delete from OS
	return wg.deleteInterface(ii.IfName)
}
