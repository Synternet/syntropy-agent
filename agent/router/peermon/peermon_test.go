package peermon

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/peerlist"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/routeselector"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
)

// These are integral tests.
// Internal parts should be tested in unit tests

func TestPeerMonitorSpeedStrategy(t *testing.T) {
	cfg := routeselector.RouteSelectorConfig{
		AverageSize:              24,
		RouteStrategy:            config.RouteStrategySpeed,
		RerouteRatio:             1.1,
		RerouteDiff:              10,
		RouteDeleteLossThreshold: 0,
	}
	pm := New(&cfg, 1)
	addr1 := generateIP(1)
	addr2 := generateIP(2)
	addr3 := generateIP(3)
	addr4 := generateIP(4)

	addNode := func(ip netip.Prefix) {
		ifname := "SYNTROPY_" + ip.Addr().String()
		if ip == addr1 {
			ifname = "SYNTROPY_PUBLIC"
		}
		pm.AddNode(ifname, "PublicKey", ip, 0, false)
	}

	apply := func() {
		pm.peerList.Iterate(func(ip netip.Prefix, peer *peerlist.PeerInfo) {
			peer.ResetFlags()
		})
	}

	addStats := func(endpoint netip.Prefix, count int, latency, loss float32) {
		peer, ok := pm.peerList.GetPeer(endpoint)
		if !ok {
			return
		}
		for i := 0; i < count; i++ {
			peer.Add(latency, loss)
		}
	}

	fillStats := func(endpoint netip.Prefix, latency, loss float32) {
		addStats(endpoint, int(cfg.AverageSize), latency, loss)
	}

	addNode(addr1)
	addNode(addr2)
	addNode(addr3)
	addNode(addr4)
	apply()

	// Lower loss is always must
	fillStats(addr1, 100, 0.02)
	fillStats(addr2, 145, 0.11)
	fillStats(addr3, 500, 0)
	fillStats(addr4, 105, 0.3)
	best := pm.BestPath()
	if best.IP != addr3.Addr() {
		t.Errorf("Lowest loss test failed %s", best.IP)
	}

	// Test without thresholds
	cfg.RerouteDiff = 0
	cfg.RerouteRatio = 1
	fillStats(addr1, 100, 0)
	fillStats(addr2, 145, 0)
	fillStats(addr3, 250, 0)
	fillStats(addr4, 95, 0)
	best = pm.BestPath()
	if best.IP != addr4.Addr() {
		t.Errorf("Test without threshold %s", best.IP)
	}

	// decrement latency a little and calculate best
	addStats(addr1, 2, 10, 0) // latency ~92
	best = pm.BestPath()
	if best.IP != addr1.Addr() {
		t.Errorf("Test with decreased latency %s", best.IP)
	}

	// Set too big thresholds and test
	cfg.RerouteDiff = 10
	cfg.RerouteRatio = 1.05
	// Increment latency close to threshold
	fillStats(addr1, 100, 0)
	best = pm.BestPath()
	if best.IP != addr1.Addr() {
		t.Errorf("Test with too big threshold %s", best.IP)
	}

	// Correct threshold test
	cfg.RerouteDiff = 5
	cfg.RerouteRatio = 1.05
	best = pm.BestPath()
	if best.IP != addr4.Addr() {
		t.Errorf("Test with correct threshold %s", best.IP)
	}

	// try reducing to threshold limits
	addStats(addr1, 2, 1, 0) // latency ~91
	best = pm.BestPath()
	if best.IP != addr4.Addr() {
		t.Errorf("Test with latency above threshold %s", best.IP)
		peer, _ := pm.peerList.GetPeer(addr1)
		t.Error(peer.Latency())
	}
	// reach the limit
	addStats(addr1, 1, 1, 0) // latency ~87
	best = pm.BestPath()
	if best.IP != addr1.Addr() {
		t.Errorf("Test with latency reach threshold %s", best.IP)
	}

	// test incomplete statistics
	peer, ok := pm.peerList.GetPeer(addr4)
	if !ok {
		t.Errorf("Expected peer not found")
	}
	peer.Add(0, 0)

	best = pm.BestPath()
	if best.IP != addr1.Addr() {
		t.Errorf("Test with incomplete statistics %s", best.IP)
		t.Error(peer.Valid(), peer.StatsIncomplete(), peer.Latency())
	}
}

func TestPeerMonitorDirectRouteStrategy(t *testing.T) {
	cfg := routeselector.RouteSelectorConfig{
		AverageSize:              24,
		RouteStrategy:            config.RouteStrategyDirectRoute,
		RerouteRatio:             1.1,
		RerouteDiff:              10,
		RouteDeleteLossThreshold: 0,
	}
	pm := New(&cfg, 2)

	addr1 := generateIP(1)
	addr2 := generateIP(2)
	addr3 := generateIP(3)
	addr4 := generateIP(4)

	addNode := func(ip netip.Prefix) {
		ifname := "SYNTROPY_" + ip.Addr().String()
		if ip == addr1 {
			ifname = "SYNTROPY_PUBLIC"
		}
		pm.AddNode(ifname, "PublicKey", ip, 0, false)
	}

	apply := func() {
		pm.peerList.Iterate(func(ip netip.Prefix, peer *peerlist.PeerInfo) {
			peer.ResetFlags()
		})
	}

	addStats := func(endpoint netip.Prefix, count int, latency, loss float32) {
		peer, ok := pm.peerList.GetPeer(endpoint)
		if !ok {
			return
		}
		for i := 0; i < count; i++ {
			peer.Add(latency, loss)
		}
	}

	fillStats := func(endpoint netip.Prefix, latency, loss float32) {
		addStats(endpoint, int(cfg.AverageSize), latency, loss)
	}

	addNode(addr1)
	addNode(addr2)
	addNode(addr3)
	addNode(addr4)
	apply()

	// Lower loss is always must
	fillStats(addr1, 100, 0.02)
	fillStats(addr2, 145, 0.11)
	fillStats(addr3, 500, 0)
	fillStats(addr4, 105, 0.3)
	best := pm.BestPath()
	if best.IP != addr3.Addr() {
		t.Errorf("Lowest loss test failed %s", best.IP)
	}

	// Test without thresholds
	cfg.RerouteDiff = 0
	cfg.RerouteRatio = 1
	fillStats(addr1, 100, 0)
	fillStats(addr2, 145, 0)
	fillStats(addr3, 250, 0)
	fillStats(addr4, 95, 0)
	best = pm.BestPath()
	if best.IP != addr4.Addr() {
		t.Errorf("Test without threshold %s", best.IP)
	}

	// Too small threshold - no change
	cfg.RerouteDiff = 3
	cfg.RerouteRatio = 1.03
	best = pm.BestPath()
	if best.IP != addr4.Addr() {
		t.Errorf("Test with too small threshold %s", best.IP)
	}

	// Threshold hit - use public
	cfg.RerouteDiff = 6
	cfg.RerouteRatio = 1.05
	best = pm.BestPath()
	if best.IP != addr1.Addr() {
		t.Errorf("Test with correct threshold %s", best.IP)
	}

	// decrement latency a little and calculate best
	addStats(addr1, 1, 200, 0) // latency ~104
	best = pm.BestPath()
	if best.IP != addr4.Addr() {
		t.Errorf("Test with increased latency %s", best.IP)
	}

	// Reduce latency close to threshold
	addStats(addr1, 2, 20, 0) // latency ~97
	best = pm.BestPath()
	if best.IP != addr1.Addr() {
		t.Errorf("Test with too big threshold %s", best.IP)
	}

	// test incomplete statistics
	peer, ok := pm.peerList.GetPeer(addr4)
	if !ok {
		t.Errorf("Expected peer not found")
	}
	peer.Add(0, 0)

	best = pm.BestPath()
	if best.IP != addr1.Addr() {
		t.Errorf("Test with incomplete statistics %s", best.IP)
		t.Error(peer.Valid(), peer.StatsIncomplete(), peer.Latency())
	}
}

func generateIP(i int) netip.Prefix {
	ip := netip.MustParseAddr(fmt.Sprintf("10.10.10.%d", i))
	return netip.PrefixFrom(ip, ip.BitLen())
}
