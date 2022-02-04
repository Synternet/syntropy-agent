package mole

import (
	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/agent/router"
	"github.com/SyntropyNet/syntropy-agent/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

// Some compatibility layer to directly access apprentice
func (m *Mole) Wireguard() *swireguard.Wireguard {
	return m.wg
}

// Some compatibility layer to directly access apprentice
func (m *Mole) Router() *router.Router {
	return m.router
}

// Close and cleanup
func (m *Mole) Close() error {
	// Delete host routes to peers.
	// These are routes added to connected WG peers via original default gateway.
	// NOTE: other routes will be deleted on interface destroy
	if config.CleanupOnExit() {
		for _, entry := range m.cache.peers {
			if entry.gwIfname != "" {
				logger.Debug().Println(pkgName, "Cleanup peer host route", entry.destIP,
					"on", entry.gwIfname)
				err := netcfg.RouteDel(entry.gwIfname, entry.destIP)
				if err != nil {
					// Warning and try to continue.
					logger.Warning().Println(pkgName, "peer host route cleanup", err)
				}
			}
		}
	}

	return m.wg.Close()
}

// Flush old cache (prepare to build new cache)
func (m *Mole) Flush() {
	m.wg.Flush()
}

// Apply pending results (sync cache to reality)
// Send some messages to controller (Writter), if needed
func (m *Mole) Apply() {
	err := m.wg.Apply()
	if err != nil {
		logger.Error().Println(pkgName, "wireguard apply", err)
	}

	routeStatusMessage := routestatus.New()
	peersActiveDataMessage := peeradata.NewMessage()

	routeRes, peersData := m.router.Apply()

	routeStatusMessage.Add(routeRes...)
	peersActiveDataMessage.Add(peersData...)

	routeStatusMessage.Send(m.writer)
	peersActiveDataMessage.Send(m.writer)
}
