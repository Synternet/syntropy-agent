package swireguard

import (
	"fmt"

	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/internal/netfilter"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/netcfg"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type InterfaceInfo struct {
	IfName     string
	PublicKey  string
	privateKey string
	IP         string
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
		myDev.flushPeers()
	}

	if osDev == nil {
		// create interface, if missing
		logger.Info().Println(pkgName, "create interface", ii.IfName)
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
		logger.Info().Println(pkgName, "reusing existing interface", ii.IfName)
		privKey = osDev.PrivateKey
		if isPortInRange(osDev.ListenPort) {
			port = osDev.ListenPort
		} else {
			port = findFreePort(ii.Port)
		}
	}

	if mtu := config.GetInterfaceMTU(); mtu > 0 {
		err = netcfg.InterfaceSetMTU(ii.IfName, mtu)
		if err != nil {
			logger.Error().Println(pkgName, "MTU error: ", ii.IfName, mtu, err)
		}
	}

	wgconf := wgtypes.Config{
		PrivateKey: &privKey,
		ListenPort: &port,
	}

	err = wg.wgc.ConfigureDevice(ii.IfName, wgconf)
	if err != nil {
		return fmt.Errorf("configure interface failed: %s", err.Error())
	}

	err = netcfg.InterfaceUp(ii.IfName)
	if err != nil {
		logger.Error().Println(pkgName, "Could not up interface: ", ii.IfName, err)
	}
	if !netcfg.InterfaceHasIP(ii.IfName, ii.IP) {
		err = netcfg.InterfaceIPAdd(ii.IfName, ii.IP)
		if err != nil {
			logger.Error().Println(pkgName, "Could not set IP address: ", ii.IfName, err)
		}
	}
	// Why this config variale configures only forward, and does not impact other iptables rules ???
	if config.CreateIptablesRules() {
		err = netfilter.ForwardEnable(ii.IfName)
		if err != nil {
			logger.Error().Println(pkgName, "netfilter forward enable", ii.IfName, err)
		}
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

	dev := wg.Device(ii.IfName)
	if dev == nil {
		logger.Warning().Println(pkgName, "Cannot remove non-existing interface ", ii.IfName)
		return nil
	}

	// Delete from cache
	wg.interfaceCacheDel(dev)
	// delete from OS
	return wg.deleteInterface(ii.IfName)
}
