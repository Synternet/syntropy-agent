// ctrlmgr is Controler Host Routes manager
package ctrlmgr

import (
	"net"
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

// In case a VPN client is used sometimes may happen that other peer is stopped
// Or when deleting connection - VPN server first deletes connection
// In such situation client is left in nonet situation and will not get any more messages from controller
// To workarround such case - add direct host routes to cloud controller (only in VPN_CLIENT=true)

const pkgName = "ControlerHostRoutes. "

type ControllerHostRouteManager struct {
	gw     netip.Addr
	ifname string
	routes []netip.Prefix
}

func (chr *ControllerHostRouteManager) Init() error {
	// Add direct route to cloud controller in case VPN_CLIENT is enabled
	if !config.IsVPNClient() {
		return nil
	}
	// Host routes are needed only for cloud controller
	if config.GetControllerType() != config.ControllerSaas {
		return nil
	}

	logger.Info().Println(pkgName, "Create hostroutes to cloud controller")

	addrs, err := net.LookupIP(config.GetCloudURL())
	if err != nil {
		logger.Error().Println(pkgName, "Could not resolve cloud URL", err)
		return err
	}

	chr.gw, chr.ifname, err = netcfg.DefaultRoute()
	if err != nil {
		logger.Error().Println(pkgName, "Could not find default route", err)
		return err
	}

	for _, addr := range addrs {
		ipAddr, ok := netip.AddrFromSlice(addr)
		if !ok {
			continue
		}
		// Agent controlls only IPv4 addresses
		// Ignore IPv6 for now
		if !ipAddr.Is4() {
			continue
		}
		dest := netip.PrefixFrom(ipAddr, ipAddr.BitLen())

		logger.Debug().Println(pkgName, "Controller route add", dest, "via", chr.gw, chr.ifname)
		err = netcfg.RouteAdd(chr.ifname, &chr.gw, &dest)
		if err != nil {
			logger.Warning().Println(pkgName, "hostroute", dest, "error", err)
			continue
		}

		chr.routes = append(chr.routes, dest)
	}

	return nil
}

func (chr *ControllerHostRouteManager) Close() error {
	// Delete (if created) host route to controller
	// (It was enabled in VPN_CLIENT=true case)
	// NOTE: caller should control locking

	if len(chr.routes) == 0 {
		// No routes were added - no need to delete
		return nil
	}

	logger.Info().Println(pkgName, "Cleanup host routes to cloud controller")
	for _, c := range chr.routes {
		logger.Debug().Println(pkgName, "cleanup", c, "on", chr.ifname)
		err := netcfg.RouteDel(chr.ifname, &c)
		if err != nil {
			// Warning and try to continue.
			logger.Warning().Println(pkgName, "host route cleanup", err)
		}
	}

	// cleanup controller IPs
	chr.routes = []netip.Prefix{}

	return nil
}
