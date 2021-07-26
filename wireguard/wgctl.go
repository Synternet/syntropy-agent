package wireguard

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/SyntropyNet/syntropy-agent-go/config"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type InterfaceInfo struct {
	IfName    string
	IP        string
	PublicKey string
	Port      int
}

type PeerInfo struct {
}

func (wg *Wireguard) getPrivateKey(ifname string) (key wgtypes.Key, err error) {
	privateFileName := config.AgentConfigDir + "/privatekey-" + ifname
	publicFileName := config.AgentConfigDir + "/publickey-" + ifname

	// at first try to read cached key
	strKey, err := ioutil.ReadFile(privateFileName)
	if err == nil {
		key, err = wgtypes.ParseKey(string(strKey))
		if err != nil {
			log.Println("parse key error: ", err)
			// Could not parse key. Most probably cache file is corrupted.
			// Do not exit and create a new key (continue to new key generation fallback)
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
			log.Println("Caching private key error: ", err)
		}
		// TODO: do I really need to cache public key ??
		err = ioutil.WriteFile(publicFileName, []byte(key.PublicKey().String()), 0600)
		if err != nil {
			log.Println("Caching public key error: ", err)
		}
	}

	return key, nil
}

func (wg *Wireguard) InterfaceExist(ifname string) bool {
	wgdevs, err := wg.Devices()
	if err != nil {
		log.Println("wgctrl.Devices: ", err)
		return false
	}
	for _, w := range wgdevs {
		if ifname == w.Name {
			return true
		}
	}
	return false
}

func (wg *Wireguard) CreateInterface(ii *InterfaceInfo) error {
	if ii == nil {
		return fmt.Errorf("invalid parameters to CreateInterface")
	}

	if wg.InterfaceExist(ii.IfName) {
		log.Println("Skipping existing interface ", ii.IfName)
		return nil
	}

	err := createInterfaceCmd(ii.IfName)
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

	cfg := wgtypes.Config{
		PrivateKey: &privKey,
		ListenPort: &ii.Port,
	}
	err = wg.ConfigureDevice(ii.IfName, cfg)
	if err != nil {
		return fmt.Errorf("configure interface failed: %s", err.Error())
	}

	setInterfaceUpCmd(ii.IfName)
	setInterfaceIPCmd(ii.IfName, ii.IP)

	dev, err := wg.Device(ii.IfName)
	if err != nil {
		return fmt.Errorf("reading wg device info error: %s", err.Error())
	}

	// Overwrite fields, that might have changed
	ii.Port = dev.ListenPort
	ii.PublicKey = dev.PublicKey.String()

	return nil
}

func (wg *Wireguard) RemoveInterface(ii *InterfaceInfo) error {
	if ii == nil {
		return fmt.Errorf("invalid parameters to RemoveInterface")
	}

	if !wg.InterfaceExist(ii.IfName) {
		log.Println("Cannot remove non-existing interface ", ii.IfName)
		return nil
	}

	return deleteInterfaceCmd(ii.IfName)
}

func (wg *Wireguard) AddPeer(pi *PeerInfo) error {
	return nil
}

func (wg *Wireguard) RemovePeer(pi *PeerInfo) error {
	return nil
}
