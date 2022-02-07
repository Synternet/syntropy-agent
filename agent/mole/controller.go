package mole

import (
	"net"

	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

// In case a VPN client is used sometimes may happen that other peer is stopped
// Or when deleting connection - VPN server first deletes connection
// In such situation client is left in nonet situation and will not get any more messages from controller
// To workarround such case - add direct host routes to cloud controller (only in VPN_CLIENT=true)
func (m *Mole) initControllerRoutes() {
	// Add direct route to cloud controller in case VPN_CLIENT is enabled
	if !config.IsVPNClient() {
		return
	}
	if config.GetControllerType() != config.ControllerSaas {
		return
	}

	addrs, err := net.LookupIP(config.GetCloudURL())
	if err != nil {
		logger.Error().Println(pkgName, "Resolving cloud URL", err)
		return
	}

	gw, ifname, err := netcfg.DefaultRoute()
	if err != nil {
		logger.Error().Println(pkgName, "default route", err)
		return
	}

	for _, ip := range addrs {
		// Agent controlls only IPv4 addresses
		// Ignore IPv6 for now
		if ip.To4() == nil {
			continue
		}
		ipStr := ip.String() + "/32"
		logger.Info().Println(pkgName, "Controller route add", ipStr, "via", gw, ifname)
		err = netcfg.RouteAdd(ifname, gw, ipStr)
		if err != nil {
			logger.Warning().Println(pkgName, "add hostroute to controller", ipStr, "via", gw, ifname, err)
			continue
		}
		m.cache.controller = append(m.cache.controller, peerCacheEntry{
			destIP:   ipStr,
			gateway:  gw,
			gwIfname: ifname})
	}
}

// Delete (if created) host route to controller
// (It was enabled in VPN_CLIENT=true case)
// NOTE: caller should controll locking
func (m *Mole) cleanupControllerRoutes() {
	for _, c := range m.cache.controller {
		logger.Debug().Println(pkgName, "Cleanup controller host route",
			c.destIP, "on", c.gwIfname)
		err := netcfg.RouteDel(c.gwIfname, c.destIP)
		if err != nil {
			// Warning and try to continue.
			logger.Warning().Println(pkgName, "controller host route cleanup", err)
		}
	}
	// cleanup controller IPs
	m.cache.controller = nil
}
