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

func generateIP(i int) string {
	return fmt.Sprintf("1.1.1.%d", i+1)
}

func (pm *PeerMonitor) fillStats(index int, latency, loss float32) {
	var ifname string
	if index == 0 {
		ifname = "SYNTROPY_PUBLIC"
	} else {
		ifname = fmt.Sprintf("SYNTROPY_SDN%d", index)
	}
	ip := netip.MustParseAddr(generateIP(index))

	pm.AddNode(ifname, "PublicKey", ip, index)
	for _, peer := range pm.peerList {
		if peer.ip == ip {
			for i := 0; i < int(pm.config.AverageSize); i++ {
				peer.Add(latency, loss)
			}
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
		/*
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
		*/
		{
			latency:     [4]float32{100, 90, 85, 82},
			bestLatency: 3,
			bestPublic:  0,
		},
	}

	for testIndex, test := range testData {
		pm := New(&cfg)

		for i, latency := range test.latency {
			pm.fillStats(i, latency, 0)
		}

		best, _ := bestPathLowestLatency(pm)
		if best != test.bestLatency {
			t.Errorf("Lowest latency test %d failed (%d vs %d)", testIndex, best, test.bestLatency)
		}

		best, _ = bestPathPreferPublic(pm)
		if best != test.bestPublic {
			t.Errorf("Lowest prefer public test %d failed (%d vs %d)", testIndex, best, test.bestPublic)
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
		pm := New(&cfg)

		for i, latency := range test.latency {
			pm.fillStats(i, latency, 0)
		}

		for j := 0; j < pathsCount; j++ {
			pm.lastBest = j

			best, _ := bestPathLowestLatency(pm)
			if best != test.bestLatency[j] {
				t.Errorf("Lowest latency test %d/%d failed (%d vs %d)", testIndex, j, best, test.bestLatency[j])
			}

			best, _ = bestPathPreferPublic(pm)
			if best != test.bestPublic[j] {
				t.Errorf("Lowest prefer public test %d/%d failed (%d vs %d)", testIndex, j, best, test.bestPublic[j])
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
	pm := New(&cfg)

	pm.fillStats(0, 100, 0)
	pm.fillStats(1, 90, 0)
	pm.fillStats(2, 95, 0)
	pm.fillStats(3, 95, 0)

	cfg.RouteStrategy = config.RouteStrategySpeed
	best := pm.BestPath()
	if best == nil {
		t.Errorf("Best path is nil")
	}
	if best.IP.String() != generateIP(1) {
		t.Errorf("Route speed strategy failed")
	}

	cfg.RouteStrategy = config.RouteStrategyDirectRoute
	best = pm.BestPath()
	if best == nil {
		t.Errorf("Best path is nil")
	}
	if best.IP.String() != generateIP(1) {
		t.Errorf("Route direct route strategy failed")
	}
}
