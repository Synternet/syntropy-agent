package peermon

import (
	"net/netip"
	"testing"

	"github.com/SyntropyNet/syntropy-agent/internal/config"
)

func TestPeerMonitor(t *testing.T) {
	cfg := PeerMonitorConfig{
		AverageSize:              24,
		RouteStrategy:            config.RouteStrategySpeed,
		RerouteRatio:             1.1,
		RerouteDiff:              10,
		RouteDeleteLossThreshold: 0,
	}
	pm := New(&cfg)

	addNode := func(ip netip.Prefix) {
		pm.AddNode("ifname", "PublicKey", ip, 0, false)
	}

	apply := func() {
		for _, peer := range pm.peerList {
			peer.flags = pifNone
		}
	}

	fillStats := func(endpoint netip.Prefix, latency, loss float32) {
		peer, ok := pm.peerList[endpoint]
		if !ok {
			return
		}
		for i := 0; i < int(cfg.AverageSize); i++ {
			peer.Add(latency, loss)
		}
	}

	addNode(generateIP(0))
	addNode(generateIP(1))
	addNode(generateIP(2))
	addNode(generateIP(3))
	apply()
	pm.lastBest = invalidBest()

	// Lower loss is always must
	fillStats(generateIP(0), 100, 0.02)
	fillStats(generateIP(1), 145, 0.11)
	fillStats(generateIP(2), 500, 0)
	fillStats(generateIP(3), 105, 0.3)
	best := pm.BestPath()
	if best.IP != generateIP(2).Addr() {
		t.Errorf("Lowest loss test failed %s", best.IP)
	}

	// Test without thresholds
	cfg.RerouteDiff = 0
	cfg.RerouteRatio = 1
	pm.lastBest = generateIP(0)
	fillStats(generateIP(0), 100, 0)
	fillStats(generateIP(1), 145, 0)
	fillStats(generateIP(2), 250, 0)
	fillStats(generateIP(3), 95, 0)
	best = pm.BestPath()
	if best.IP != generateIP(3).Addr() {
		t.Errorf("Test without threshold %s", best.IP)
	}

	// Set thresholds and test
	cfg.RerouteDiff = 10
	cfg.RerouteRatio = 1.05
	pm.lastBest = generateIP(0)
	best = pm.BestPath()
	if best.IP != generateIP(0).Addr() {
		t.Errorf("Test with too big threshold %s", best.IP)
	}

	cfg.RerouteDiff = 5
	cfg.RerouteRatio = 1.05
	pm.lastBest = invalidBest()
	best = pm.BestPath()
	if best.IP != generateIP(3).Addr() {
		t.Errorf("Test with correct threshold %s", best.IP)
	}

	// test incomplete statistics
	pm.lastBest = generateIP(0)
	pm.peerList[generateIP(3)].Add(0, 0)
	best = pm.BestPath()
	if best.IP != generateIP(0).Addr() {
		t.Errorf("Test with incomplete statistics %s", best.IP)
	}
}
