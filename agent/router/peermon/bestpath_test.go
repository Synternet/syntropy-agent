package peermon

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/SyntropyNet/syntropy-agent/internal/config"
)

const pathsCount = 4

type testEntry1 struct {
	latency     [pathsCount]float32
	bestLatency int
	bestPublic  int
}

type testEntry2 struct {
	latency     [pathsCount]float32
	bestLatency [pathsCount]int
	bestPublic  [pathsCount]int
}

func generateIP(i int) netip.Prefix {
	ip := netip.MustParseAddr(fmt.Sprintf("1.1.1.%d", i+1))
	return netip.PrefixFrom(ip, ip.BitLen())
}

func (pm *PeerMonitor) fillStats(index int, latency, loss float32) {
	var ifname string
	if index == 0 {
		ifname = "SYNTROPY_PUBLIC"
	} else {
		ifname = fmt.Sprintf("SYNTROPY_SDN%d", index)
	}

	ip := generateIP(index)
	pm.AddNode(ifname, "PublicKey", ip, index, false)

	peer, ok := pm.peerList[ip]
	if ok {
		peer.flags = pifNone
		for i := 0; i < int(pm.config.AverageSize); i++ {
			peer.Add(latency, loss)
		}
	}
}

// This tests the best function without previous values
func TestBestFunctions1(t *testing.T) {
	cfg := PeerMonitorConfig{
		AverageSize:              10,
		RouteStrategy:            config.RouteStrategySpeed,
		RerouteRatio:             1.1,
		RerouteDiff:              20,
		RouteDeleteLossThreshold: 0,
	}

	testData := []testEntry1{
		{
			latency:     [4]float32{20, 500, 300, 35},
			bestLatency: 0,
			bestPublic:  0,
		},
		{
			latency:     [4]float32{500, 1000, 1000, 200},
			bestLatency: 3,
			bestPublic:  3,
		},
		{
			latency:     [4]float32{100, 105, 110, 120},
			bestLatency: 0,
			bestPublic:  0,
		},
		{
			latency:     [4]float32{100, 110, 85, 78},
			bestLatency: 3,
			bestPublic:  3,
		},
		{
			latency:     [4]float32{100, 90, 85, 82},
			bestLatency: 3,
			bestPublic:  0,
		},
	}

	for testIndex, test := range testData {
		pm := New(&cfg, 1)

		for i, latency := range test.latency {
			pm.fillStats(i, latency, 0)
		}

		best, _ := bestPathLowestLatency(pm)
		if best != generateIP(test.bestLatency) {
			t.Errorf("Lowest latency test %d failed (%s vs %s)", testIndex, best, generateIP(test.bestLatency))
		}

		best, _ = bestPathPreferPublic(pm)
		if best != generateIP(test.bestPublic) {
			t.Errorf("Lowest prefer public test %d failed (%s vs %s)", testIndex, best, generateIP(test.bestPublic))
		}
	}

}

// Tests best functions and takes into account previous values
func TestBestFunctions2(t *testing.T) {
	cfg := PeerMonitorConfig{
		AverageSize:              10,
		RouteStrategy:            config.RouteStrategySpeed,
		RerouteRatio:             1.1,
		RerouteDiff:              20,
		RouteDeleteLossThreshold: 0,
	}

	testData := []testEntry2{
		{
			latency:     [pathsCount]float32{20, 500, 300, 35},
			bestLatency: [pathsCount]int{0, 0, 0, 3},
			bestPublic:  [pathsCount]int{0, 0, 0, 0},
		},
		{
			latency:     [pathsCount]float32{500, 1000, 1000, 200},
			bestLatency: [pathsCount]int{3, 3, 3, 3},
			bestPublic:  [pathsCount]int{3, 3, 3, 3},
		},
		{
			latency:     [pathsCount]float32{100, 105, 110, 120},
			bestLatency: [pathsCount]int{0, 1, 2, 0},
			bestPublic:  [pathsCount]int{0, 0, 0, 0},
		},
		{
			latency:     [pathsCount]float32{100, 110, 85, 78},
			bestLatency: [pathsCount]int{3, 3, 2, 3},
			bestPublic:  [pathsCount]int{3, 3, 3, 3},
		},
		{
			latency:     [pathsCount]float32{100, 90, 85, 82},
			bestLatency: [pathsCount]int{0, 1, 2, 3},
			bestPublic:  [pathsCount]int{0, 0, 0, 0},
		},
		{
			latency:     [pathsCount]float32{100, 105, 95, 100},
			bestLatency: [pathsCount]int{0, 1, 2, 3},
			bestPublic:  [pathsCount]int{0, 0, 0, 0},
		},
	}

	for testIndex, test := range testData {
		pm := New(&cfg, 1)

		for i, latency := range test.latency {
			pm.fillStats(i, latency, 0)
		}

		for j := 0; j < pathsCount; j++ {
			pm.lastBest = generateIP(j)

			best, _ := bestPathLowestLatency(pm)
			if best != generateIP(test.bestLatency[j]) {
				t.Errorf("Lowest latency test %d/%d failed (%s vs %s)", testIndex, j, best, generateIP(test.bestLatency[j]))
			}

			best, _ = bestPathPreferPublic(pm)
			if best != generateIP(test.bestPublic[j]) {
				t.Errorf("Lowest prefer public test %d/%d failed (%s vs %s)", testIndex, j, best, generateIP(test.bestPublic[j]))
			}
		}
	}

}

func TestBestConfiguration(t *testing.T) {
	cfg := PeerMonitorConfig{
		AverageSize:              10,
		RerouteRatio:             1.1,
		RerouteDiff:              20,
		RouteDeleteLossThreshold: 0,
	}
	pm := New(&cfg, 2)

	pm.fillStats(0, 100, 0)
	pm.fillStats(1, 90, 0)
	pm.fillStats(2, 95, 0)
	pm.fillStats(3, 95, 0)

	cfg.RouteStrategy = config.RouteStrategySpeed
	best := pm.BestPath()
	if best == nil {
		t.Errorf("Best path is nil")
	}
	if best.IP != generateIP(1).Addr() {
		t.Errorf("Route speed strategy failed")
	}

	cfg.RouteStrategy = config.RouteStrategyDirectRoute
	best = pm.BestPath()
	if best == nil {
		t.Errorf("Best path is nil")
	}
	if best.IP != generateIP(1).Addr() {
		t.Errorf("Route direct route strategy failed")
	}
}
