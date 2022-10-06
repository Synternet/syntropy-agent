package hostroute

import (
	"fmt"
	"net/netip"
	"sync"

	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

const pkgName = "HostRoute. "

type HostRouter struct {
	sync.Mutex
	gw     netip.Addr // Default gateway
	ifname string     // Interface where default gw is reachable
	routes map[netip.Prefix]*routeEntry
}

// TODO: in future need to handle IP/gw changes and reapply configuration
func (hr *HostRouter) getDefaultRoute() error {
	var err error
	hr.gw, hr.ifname, err = netcfg.DefaultRoute()
	if err != nil {
		logger.Error().Println(pkgName, "Could not find default route", err)
	}
	return err
}

func (hr *HostRouter) Init() error {

	hr.Lock()
	defer hr.Unlock()

	hr.routes = make(map[netip.Prefix]*routeEntry)
	return hr.getDefaultRoute()
}

// Adds host route IPs to cache
func (hr *HostRouter) Add(addrs ...netip.Prefix) error {
	hr.Lock()
	defer hr.Unlock()

	for _, ip := range addrs {
		e, ok := hr.routes[ip]

		if !ok {
			e = newEntry()
			hr.routes[ip] = e
		}

		e.count++
	}
	return nil
}

func (hr *HostRouter) Del(addrs ...netip.Prefix) error {
	hr.Lock()
	defer hr.Unlock()

	errCount := 0
	for _, ip := range addrs {
		e, ok := hr.routes[ip]

		if !ok {
			logger.Warning().Println(pkgName, ip, "not found")
			errCount++
			continue
		}

		if e.count == 0 {
			// This case should never happen, but have check in case of code errors
			logger.Error().Println(pkgName, "Invalid counter for", ip)
		} else {
			e.count--
		}
	}

	if errCount > 0 {
		return fmt.Errorf("not found %d", errCount)
	}
	return nil
}

func (hr *HostRouter) Apply() error {
	hr.Lock()
	defer hr.Unlock()

	if !hr.gw.IsValid() {
		err := hr.getDefaultRoute()
		if err != nil {
			return err
		}
	}

	var delIPs []netip.Prefix
	errCount := 0
	// Apply pending operations
	for ip, entry := range hr.routes {
		if entry.count == 0 {
			// Nobody needs it and entry is ready for deletion
			delIPs = append(delIPs, ip)
			// Delete the entry (if it was applied earlier)
			if !entry.pending {
				logger.Debug().Println(pkgName, "Peer host route del to",
					ip, "via", hr.ifname)
				err := netcfg.RouteDel(hr.ifname, &ip)
				if err != nil {
					// Warning and try to continue.
					logger.Warning().Println(pkgName, "peer host route delete", err)
					errCount++
				}
			}
		} else {
			// if pending==false - this entry was already applied
			if entry.pending {
				logger.Debug().Println(pkgName, "Peer host route add to", ip,
					"via", hr.gw, hr.ifname)
				err := netcfg.RouteAdd(hr.ifname, &hr.gw, &ip)
				if err != nil {
					// Add peer host route failed. It should be some route conflict.
					// In normal case this should not happen.
					// But this is not a fatal error, so I try to warn and continue.
					logger.Warning().Println(pkgName, "adding peer host route", err)
					errCount++
				}
				entry.pending = false
			}
		}
	}

	// Remove deleted entries from cache
	for _, ip := range delIPs {
		delete(hr.routes, ip)
	}

	if errCount > 0 {
		return fmt.Errorf("%d errors applying host routes", errCount)
	}
	return nil
}

// Flush is used for smart merge on newly received ConfigInfo message
// It resets counter. And entries will be removed in apply
func (hr *HostRouter) Flush() {
	hr.Lock()
	defer hr.Unlock()

	for _, entry := range hr.routes {
		entry.count = 0
	}
}

func (hr *HostRouter) Close() error {
	hr.Lock()
	defer hr.Unlock()

	// Delete host routes to peers.
	// These are routes added to connected WG peers via original default gateway.
	// And routes to controller, in VPN client case
	// They are only deleted if SYNTROPY_CLEANUP_ON_EXIT=true
	if config.CleanupOnExit() {
		for ip, entry := range hr.routes {
			// delete all added (not pending) routes
			if !entry.pending {
				logger.Debug().Println(pkgName, "Cleanup host route", ip, "via", hr.ifname)
				err := netcfg.RouteDel(hr.ifname, &ip)
				if err != nil {
					// Warning and try to continue.
					logger.Warning().Println(pkgName, "peer host route cleanup", err)
				}
			}
		}
	}

	// delete all cache entries
	hr.routes = make(map[netip.Prefix]*routeEntry)

	return nil
}
