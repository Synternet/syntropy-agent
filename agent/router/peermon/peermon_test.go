package peermon

import (
	"net/netip"
	"testing"

	"github.com/SyntropyNet/syntropy-agent/internal/config"
)

func TestPeerMonitor(t *testing.T) {
	count := 24
	pm := New(uint(count))

	addNode := func(ip netip.Addr) {
		pm.AddNode("ifname", "PublicKey", ip, 0)
	}

	fillStats := func(endpoint netip.Addr, latency, loss float32) {
		for _, peer := range pm.peerList {
			if peer.ip == endpoint {
				for i := 0; i < count; i++ {
					peer.Add(latency, loss)
				}
			}
		}

	}

	addNode(netip.MustParseAddr("1.1.1.2"))
	addNode(netip.MustParseAddr("2.2.2.2"))
	addNode(netip.MustParseAddr("3.3.3.2"))
	addNode(netip.MustParseAddr("4.4.4.2"))
	pm.lastBest = 0

	// Lower loss is always must
	fillStats(netip.MustParseAddr("1.1.1.2"), 100, 0.02)
	fillStats(netip.MustParseAddr("2.2.2.2"), 145, 0.11)
	fillStats(netip.MustParseAddr("3.3.3.2"), 500, 0)
	fillStats(netip.MustParseAddr("4.4.4.2"), 105, 0.3)
	best := pm.BestPath()
	if best.IP != netip.MustParseAddr("3.3.3.2") {
		t.Errorf("Lowest loss test failed %s", best.IP)
	}

	// Test without thresholds
	config.SetRerouteThresholds(0, 1)
	pm.lastBest = 0
	fillStats(netip.MustParseAddr("1.1.1.2"), 100, 0)
	fillStats(netip.MustParseAddr("2.2.2.2"), 145, 0)
	fillStats(netip.MustParseAddr("3.3.3.2"), 250, 0)
	fillStats(netip.MustParseAddr("4.4.4.2"), 95, 0)
	best = pm.BestPath()
	if best.IP != netip.MustParseAddr("4.4.4.2") {
		t.Errorf("Test without threshold %s", best.IP)
	}

	// Set thresholds and test
	config.SetRerouteThresholds(10, 1.05)
	pm.lastBest = 0
	best = pm.BestPath()
	if best.IP != netip.MustParseAddr("1.1.1.2") {
		t.Errorf("Test with too big threshold %s", best.IP)
	}

	config.SetRerouteThresholds(5, 1.05)
	pm.lastBest = 0
	best = pm.BestPath()
	if best.IP != netip.MustParseAddr("4.4.4.2") {
		t.Errorf("Test with correct threshold %s", best.IP)
	}

	// test incomplete statistics
	pm.lastBest = 0
	pm.peerList[3].Add(0, 0)
	best = pm.BestPath()
	if best.IP != netip.MustParseAddr("1.1.1.2") {
		t.Errorf("Test with incomplete statistics %s", best.IP)
	}
}
