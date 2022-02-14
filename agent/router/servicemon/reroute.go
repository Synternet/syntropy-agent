package servicemon

import (
	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"net"
)

func addAllowedIp(ifname string, pubkey string, ip string) {
	wgClient, _ := wgctrl.New()
	dev, err := wgClient.Device(ifname)
	if err != nil {
		return
	}
	wgconf := wgtypes.Config{}
	pcfg := &wgtypes.PeerConfig{}
	for _, peer := range dev.Peers {
		if peer.PublicKey.String() == pubkey {
			_, ipnet, _ := net.ParseCIDR(ip)
			pcfg.ReplaceAllowedIPs = true
			pcfg.UpdateOnly = true
			pcfg.AllowedIPs = append(peer.AllowedIPs, *ipnet)
			pcfg.PublicKey = peer.PublicKey
			pcfg.PersistentKeepaliveInterval = &peer.PersistentKeepaliveInterval
			wgconf.Peers = append(wgconf.Peers, *pcfg)
		}
	}
	wgClient.ConfigureDevice(ifname, wgconf)
	wgClient.Close()
}

func removeAllowedIps(ip string) {
	wgClient, _ := wgctrl.New()
	devices, err := wgClient.Devices()
	if err != nil {
		return
	}
	for _, dev := range devices {
		wgconf := wgtypes.Config{}
		pcfg := &wgtypes.PeerConfig{}
		_, ipnet, _ := net.ParseCIDR(ip)
		for _, peer := range dev.Peers {
			for _, allowed_ip := range peer.AllowedIPs {
				if allowed_ip.String() != ipnet.String() {
					pcfg.AllowedIPs = append(pcfg.AllowedIPs, allowed_ip)
				}
			}
			pcfg.ReplaceAllowedIPs = true
			pcfg.UpdateOnly = true
			pcfg.PublicKey = peer.PublicKey
			pcfg.PersistentKeepaliveInterval = &peer.PersistentKeepaliveInterval
			wgconf.Peers = append(wgconf.Peers, *pcfg)
		}
		wgClient.ConfigureDevice(dev.Name, wgconf)
	}
	wgClient.Close()
}

func (sm *ServiceMonitor) Reroute(newgw string) []*peeradata.Entry {
	peersActiveData := []*peeradata.Entry{}

	sm.Lock()
	defer sm.Unlock()

	for dest, routeList := range sm.routes {
		currRoute := routeList.GetActive()
		var newRoute *routeEntry = nil
		if newgw != peermon.NoRoute {
			newRoute = routeList.Find(newgw)
			if newRoute == nil {
				logger.Error().Println(pkgName, "New route ", newgw, "not found.")
			}
		}

		ret := routeList.Reroute(newRoute, currRoute, dest)
		if ret != nil {
			peersActiveData = append(peersActiveData, ret)
		}
	}

	return peersActiveData
}

// Reroute one routeList (aka Service Group)
func (rl *routeList) Reroute(newRoute, oldRoute *routeEntry, destination string) *peeradata.Entry {
	switch {
	case newRoute == oldRoute:
		// Nothing to change
		return nil

	case newRoute == nil:
		// Delete active route
		logger.Info().Println(pkgName, "remove route", destination, oldRoute.ifname)
		err := netcfg.RouteDel(oldRoute.ifname, destination)
		if err != nil {
			logger.Error().Println(pkgName, "could not remove route to", destination, "via", oldRoute.ifname)
		}
		// reset active flags
		oldRoute.ClearFlags(rfActive)

		// TODO Fix this. Currently Allowed ip has to be removed from every peer.
		removeAllowedIps(destination)
		// Return route change
		return peeradata.NewEntry(oldRoute.connectionID, 0, 0)

	case oldRoute == nil:
		// No previous active route was present. Set new route
		logger.Info().Println(pkgName, "add route", destination, newRoute.ifname)
		err := netcfg.RouteAdd(newRoute.ifname, "", destination)
		if err != nil {
			logger.Error().Println(pkgName, "could not add route to", destination, "via", newRoute.ifname)
		}
		// set active flags
		newRoute.SetFlag(rfActive)

		// Todo should reuse Mole for setting up allowed ip.
		addAllowedIp(newRoute.ifname, newRoute.publicKey, destination)
		// Return route change
		return peeradata.NewEntry(0, newRoute.connectionID, newRoute.groupID)

	default:
		// Change the route to new active
		logger.Info().Println(pkgName, "replace route", destination, oldRoute.ifname, "->", newRoute.ifname)
		err := netcfg.RouteReplace(newRoute.ifname, "", destination)
		if err != nil {
			logger.Error().Println(pkgName, "could not change routes to", destination, "via", newRoute.ifname)
		}
		// change active flags
		oldRoute.ClearFlags(rfActive)
		newRoute.SetFlag(rfActive)

		// Return route change
		return peeradata.NewEntry(oldRoute.connectionID, newRoute.connectionID, newRoute.groupID)
	}
}
