package swireguard

import (
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const KeepAlliveDuration = 15 * time.Second

type PeerInfo struct {
	IfName       string
	PublicKey    string
	ConnectionID int
	GroupID      int
	AgentID      int
	IP           netip.Addr
	Port         int
	Gateway      netip.Addr
	AllowedIPs   []netip.Prefix
	Stats        PeerStats
}

type ServiceInfo struct {
	IP            netip.Prefix
	ConnectionIDs []int
}

// Structure conversion helper
func (pi *PeerInfo) asPeerConfig() (*wgtypes.PeerConfig, error) {
	var err error
	pcfg := &wgtypes.PeerConfig{}

	pcfg.PublicKey, err = wgtypes.ParseKey(pi.PublicKey)
	if err != nil {
		return nil, err
	}
	if pi.IP.IsValid() && pi.Port > 0 {
		pcfg.Endpoint = &net.UDPAddr{
			IP:   pi.IP.AsSlice(),
			Port: pi.Port,
		}
	}

	for _, e := range pi.AllowedIPs {
		pcfg.AllowedIPs = append(pcfg.AllowedIPs, net.IPNet{
			IP:   e.Addr().AsSlice(),
			Mask: net.CIDRMask(e.Bits(), e.Addr().BitLen()),
		})
	}

	return pcfg, nil
}

// AddPeer adds a peer to Wireguard interface and internal cache
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

	return nil
}

// RemovePeer removes a peer from Wireguard interface and internal cache
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

	return nil
}

// applyPeers does a synchronisation from cache to OS WG interface
// adds missing, and removes residual peers
func (wg *Wireguard) applyPeers(ii *InterfaceInfo) (allowedIPs []netip.Prefix, err error) {
	dev, err := wg.wgc.Device(ii.IfName)
	if err != nil {
		return
	}

	wgconf := wgtypes.Config{
		ReplacePeers: false,
	}

	// add missing peers
	for _, myPeer := range ii.peers {
		addPeer := true
		for _, osPeer := range dev.Peers {
			if myPeer.PublicKey == osPeer.PublicKey.String() {
				addPeer = false
				break
			}
		}
		if addPeer {
			pcfg, err := myPeer.asPeerConfig()
			if err != nil {
				logger.Error().Println(pkgName, ii.IfName, "(re)apply peers", err)
				continue
			}
			wgconf.Peers = append(wgconf.Peers, *pcfg)
		}
	}

	// remove residual peers
	for _, osPeer := range dev.Peers {
		needRemove := true
		for _, myPeer := range ii.peers {
			if myPeer.PublicKey == osPeer.PublicKey.String() {
				needRemove = false
				break
			}
		}
		if needRemove {
			wgconf.Peers = append(wgconf.Peers, wgtypes.PeerConfig{
				PublicKey: osPeer.PublicKey,
				Remove:    true,
			})

			for _, aIP := range osPeer.AllowedIPs {
				addr, ok := netip.AddrFromSlice(aIP.IP)
				if !ok {
					continue
				}
				bitlen, _ := aIP.Mask.Size()
				allowedIPs = append(allowedIPs, netip.PrefixFrom(addr, bitlen))
			}
		}
	}

	// apply changes if needed
	if len(wgconf.Peers) > 0 {
		err = wg.wgc.ConfigureDevice(ii.IfName, wgconf)
	}

	return
}
