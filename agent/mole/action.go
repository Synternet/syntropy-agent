package mole

import (
	"github.com/SyntropyNet/syntropy-agent/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent/agent/router"
	"github.com/SyntropyNet/syntropy-agent/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
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
	if config.CleanupOnExit() {
		err := m.filter.Close()
		if err != nil {
			logger.Error().Println(pkgName, "iptables close", err)
		}
	}

	err := m.wg.Close()

	return err
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
