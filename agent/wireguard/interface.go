package wireguard

import (
	"fmt"

	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/netfilter"
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

	if dev := wg.Device(ii.IfName); dev != nil {
		// TODO add checking, if cached info matches required (compare PublicKeys)
		logger.Debug().Println(pkgName, "Do not (re)creating existing interface ", ii.IfName)
		return nil
	}

	err := createInterface(ii.IfName)
	if err != nil {
		return fmt.Errorf("create wg interface failed: %s", err.Error())
	}

	privKey, err := wg.getPrivateKey(ii.IfName)
	if err != nil {
		return fmt.Errorf("private key error: %s", err.Error())
	}

	if ii.Port == 0 {
		ii.Port = GetFreePort(ii.IfName)
	}

	wgconf := wgtypes.Config{
		PrivateKey: &privKey,
		ListenPort: &ii.Port,
	}
	err = wg.wgc.ConfigureDevice(ii.IfName, wgconf)
	if err != nil {
		return fmt.Errorf("configure interface failed: %s", err.Error())
	}

	err = netcfg.InterfaceUp(ii.IfName)
	if err != nil {
		logger.Error().Println(pkgName, "Could not up interface: ", ii.IfName, err)
	}
	err = netcfg.InterfaceIPAdd(ii.IfName, ii.IP)
	if err != nil {
		logger.Error().Println(pkgName, "Could not set IP address: ", ii.IfName, err)
	}
	err = netfilter.ForwardEnable(ii.IfName)
	if err != nil {
		logger.Error().Println(pkgName, "netfilter forward enable", ii.IfName, err)
	}

	dev, err := wg.wgc.Device(ii.IfName)
	if err != nil {
		return fmt.Errorf("reading wg device info error: %s", err.Error())
	}

	// Add current configuration to cache
	wg.interfaceCacheAdd(&InterfaceInfo{
		IfName:     ii.IfName,
		PublicKey:  dev.PublicKey.String(),
		privateKey: dev.PrivateKey.String(),
		Port:       dev.ListenPort,
		IP:         ii.IP,
	})

	// Overwrite fields, that might have changed
	ii.Port = dev.ListenPort
	ii.PublicKey = dev.PublicKey.String()

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
	return deleteInterface(ii.IfName)
}
