package swireguard

import (
	"fmt"
	"net"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/netfilter"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const KeepAlliveDuration = 15 * time.Second

type PeerInfo struct {
	IfName       string
	PublicKey    string
	ConnectionID int
	IP           string
	Port         int
	Gateway      string
	AllowedIPs   []string
	Stats        PeerStats
}

func (pi *PeerInfo) asPeerConfig() (*wgtypes.PeerConfig, error) {
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
		// I need only network address here, no need to "patch" parseCidr's result
		_, netip, err := net.ParseCIDR(e)
		if err == nil && netip != nil {
			pcfg.AllowedIPs = append(pcfg.AllowedIPs, *netip)
		}
	}

	return pcfg, nil
}

func (wg *Wireguard) AddPeer(pi *PeerInfo) error {
	if pi == nil {
		return fmt.Errorf("invalid parameters to AddPeer")
	}

	if !wg.interfaceExist(pi.IfName) {
		return fmt.Errorf("cannot configure non-existing interface %s", pi.IfName)
	}

	wgconf := wgtypes.Config{}
	pcfg, err := pi.asPeerConfig()
	if err != nil {
		return err
	}
	peerKeepAliveDuration := KeepAlliveDuration
	pcfg.PersistentKeepaliveInterval = &peerKeepAliveDuration
	pcfg.ReplaceAllowedIPs = true

	wgconf.Peers = append(wgconf.Peers, *pcfg)

	err = wg.wgc.ConfigureDevice(pi.IfName, wgconf)
	if err != nil {
		return fmt.Errorf("configure interface failed: %s", err.Error())
	}

	// Add peer to cache
	wg.peerCacheAdd(pi)

	// TODO: check and cleanup old obsolete rules
	if len(pcfg.AllowedIPs) > 0 {
		// NOTE: pi and pcfg actually are same data, but different format.
		// I am using IP from pcfg, since pi has CIDR notation,
		// and pcfg already parsed the data
		wg.peerMonitor.AddNode(pi.Gateway, pcfg.AllowedIPs[0].IP.String())
	}

	err = wg.router.RouteAdd(
		&common.SdnNetworkPath{
			Ifname:  pi.IfName,
			Gateway: pi.Gateway,
			ID:      pi.ConnectionID,
		}, pi.AllowedIPs...)
	if err != nil {
		logger.Error().Println(pkgName, "route add", err)
	}
	err = netfilter.RulesAdd(pi.AllowedIPs...)
	if err != nil {
		logger.Error().Println(pkgName, "iptables rules add", err)
	}

	return nil
}

func (wg *Wireguard) RemovePeer(pi *PeerInfo) error {
	if pi == nil {
		return fmt.Errorf("invalid parameters to RemovePeer")
	}

	if !wg.interfaceExist(pi.IfName) {
		return fmt.Errorf("cannot configure non-existing interface %s", pi.IfName)
	}

	// Add peer to cache
	wg.peerCacheDel(pi)

	wgconf := wgtypes.Config{}
	pcfg, err := pi.asPeerConfig()
	if err != nil {
		return err
	}
	pcfg.Remove = true
	wgconf.Peers = append(wgconf.Peers, *pcfg)

	err = wg.wgc.ConfigureDevice(pi.IfName, wgconf)
	if err != nil {
		return fmt.Errorf("configure interface failed: %s", err.Error())
	}

	err = wg.router.RouteDel(&common.SdnNetworkPath{Ifname: pi.IfName}, pi.AllowedIPs...)
	if err != nil {
		logger.Error().Println(pkgName, "route del", err)
	}
	err = netfilter.RulesDel(pi.AllowedIPs...)
	if err != nil {
		logger.Error().Println(pkgName, "iptables rules del", err)
	}

	return nil
}
