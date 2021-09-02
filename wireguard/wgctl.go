package wireguard

import (
	"fmt"
	"io/ioutil"
	"net"

	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/netfilter"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/netcfg"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type InterfaceInfo struct {
	IfName    string
	PublicKey string
	IP        string
	Port      int
}

type PeerInfo struct {
	IfName     string
	PublicKey  string
	IP         string
	Port       int
	Gateway    string
	AllowedIPs []string
}

func (pi *PeerInfo) AsPeerConfig() (*wgtypes.PeerConfig, error) {
	var err error
	pcfg := &wgtypes.PeerConfig{}

	pcfg.PublicKey, err = wgtypes.ParseKey(pi.PublicKey)
	if err != nil {
		return nil, err
	}
	if pi.IP != "" && pi.Port > 0 {
		pcfg.Endpoint = &net.UDPAddr{
			IP:   net.ParseIP(pi.IP),
			Port: pi.Port,
		}
	}

	for _, e := range pi.AllowedIPs {
		_, netip, err := net.ParseCIDR(e)
		if err == nil && netip != nil {
			pcfg.AllowedIPs = append(pcfg.AllowedIPs, *netip)
		}
	}

	return pcfg, nil
}

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

func (wg *Wireguard) InterfaceExist(ifname string) bool {
	wgdevs, err := wg.wgc.Devices()
	if err != nil {
		logger.Error().Println(pkgName, "Failed listing wireguard devices: ", err)
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
		return fmt.Errorf("netfilter forward enable error: %s", err.Error())
	}

	dev, err := wg.wgc.Device(ii.IfName)
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
		logger.Warning().Println(pkgName, "Cannot remove non-existing interface ", ii.IfName)
		return nil
	}

	return deleteInterface(ii.IfName)
}

func (wg *Wireguard) AddPeer(pi *PeerInfo) error {
	if pi == nil {
		return fmt.Errorf("invalid parameters to AddPeer")
	}

	if !wg.InterfaceExist(pi.IfName) {
		return fmt.Errorf("cannot configure non-existing interface %s", pi.IfName)
	}

	wgconf := wgtypes.Config{}
	pcfg, err := pi.AsPeerConfig()
	if err != nil {
		return err
	}
	wgconf.Peers = append(wgconf.Peers, *pcfg)

	err = wg.wgc.ConfigureDevice(pi.IfName, wgconf)
	if err != nil {
		return fmt.Errorf("configure interface failed: %s", err.Error())
	}

	// TODO: check and cleanup old obsolete rules
	if len(pcfg.AllowedIPs) > 0 {
		// NOTE: pi and pcfg actually are same data, but different format.
		// I am using IP from pcfg, since pi has CIDR notation,
		// and pcfg already parsed the data
		wg.sdn.AddNode(pi.Gateway, pcfg.AllowedIPs[0].IP.String())
	}

	err = wg.router.RouteAdd(pi.IfName, pi.Gateway, pi.AllowedIPs...)
	if err != nil {
		return fmt.Errorf("route add failed: %s", err.Error())
	}
	err = netfilter.RulesAdd(pi.AllowedIPs...)
	if err != nil {
		return fmt.Errorf("iptables rules add failed: %s", err.Error())
	}

	return nil
}

func (wg *Wireguard) RemovePeer(pi *PeerInfo) error {
	if pi == nil {
		return fmt.Errorf("invalid parameters to RemovePeer")
	}

	if !wg.InterfaceExist(pi.IfName) {
		return fmt.Errorf("cannot configure non-existing interface %s", pi.IfName)
	}

	wgconf := wgtypes.Config{}
	pcfg, err := pi.AsPeerConfig()
	if err != nil {
		return err
	}
	pcfg.Remove = true
	wgconf.Peers = append(wgconf.Peers, *pcfg)

	err = wg.wgc.ConfigureDevice(pi.IfName, wgconf)
	if err != nil {
		return fmt.Errorf("configure interface failed: %s", err.Error())
	}

	err = netcfg.RouteDel(pi.IfName, pi.AllowedIPs...)
	if err != nil {
		return fmt.Errorf("route add failed: %s", err.Error())
	}
	err = netfilter.RulesDel(pi.AllowedIPs...)
	if err != nil {
		return fmt.Errorf("iptables rules del failed: %s", err.Error())
	}

	return nil
}
