// ctrlmgr is Controler Host Routes manager
package ctrlmgr

import (
	"net"
	"net/netip"

	"github.com/SyntropyNet/syntropy-agent/agent/hostroute"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

// In case a VPN client is used sometimes may happen that other peer is stopped
// Or when deleting connection - VPN server first deletes connection
// In such situation client is left in nonet situation and will not get any more messages from controller
// To workarround such case - add direct host routes to cloud controller (only in VPN_CLIENT=true)

const pkgName = "ControlerHostRoutes. "

type ControllerHostRouteManager struct {
	hostRoute *hostroute.HostRouter
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
	chr.hostRoute = &hostroute.HostRouter{}
	err := chr.hostRoute.Init()
	if err != nil {
		return err
	}

	addrs, err := net.LookupIP(config.GetCloudURL())
	if err != nil {
		logger.Error().Println(pkgName, "Could not resolve cloud URL", err)
		return err
	}

	for _, addr := range addrs {
		ipAddr, ok := netip.AddrFromSlice(addr)
		if !ok {
			continue
		}
		// Agent controlls only IPv4 addresses
		// TODO: Ignore IPv6 for now.
		if !ipAddr.Is4() {
			continue
		}
		dest := netip.PrefixFrom(ipAddr, ipAddr.BitLen())

		err = chr.hostRoute.Add(dest)
		if err != nil {
			logger.Warning().Println(pkgName, "controller host route add", addr, err)
		}
	}

	return chr.hostRoute.Apply()
}

// Delete (if created) host route to controller
// (If it was enabled in VPN_CLIENT=true case)
func (chr *ControllerHostRouteManager) Close() error {
	if chr.hostRoute == nil {
		// no host routes initialised (non VPN client case)
		// nothing to delete
		return nil
	}

	if !config.CleanupOnExit() {
		// Do nothing if so configured
		return nil
	}

	logger.Info().Println(pkgName, "Cleanup host routes to cloud controller")
	return chr.hostRoute.Close()
}
