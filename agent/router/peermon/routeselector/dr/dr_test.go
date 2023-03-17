package dr

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/peerlist"
	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/routeselector"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
)

const pathsCount = 4

type testEntry struct {
	latency [pathsCount]float32
	best    [pathsCount]int
}

func generateIP(i int) netip.Prefix {
	ip := netip.MustParseAddr(fmt.Sprintf("1.1.1.%d", i+1))
	return netip.PrefixFrom(ip, ip.BitLen())
}

func (drs *directRouteSelector) fillStats(index int, latency, loss float32) {
	var ifname string
	if index == 0 {
		ifname = "SYNTROPY_PUBLIC"
	} else {
		ifname = fmt.Sprintf("SYNTROPY_SDN%d", index)
	}

	ip := generateIP(index)
	drs.peerlist.AddPeer(ifname, "PublicKey", ip, index, false)

	peer, ok := drs.peerlist.GetPeer(ip)
	if ok {
		peer.ResetFlags()
		for i := 0; i < int(drs.config.AverageSize); i++ {
			peer.Add(latency, loss)
		}
	}
}

func (drs *directRouteSelector) addStats(index int, latency, loss float32) {
	peer, ok := drs.peerlist.GetPeer(generateIP(index))
	if ok {
		peer.Add(latency, loss)
	}
}

// Tests DirectRoute selector with moment best values
// (without taking into account previous values)
func TestDirectRouteSelectorMomentary(t *testing.T) {
	cfg := routeselector.RouteSelectorConfig{
		AverageSize:              10,
		RouteStrategy:            config.RouteStrategyDirectRoute,
		RerouteRatio:             1.1,
		RerouteDiff:              10,
		RouteDeleteLossThreshold: 0,
	}

	testData := []testEntry{
		{
			// Big differences, public best
			latency: [pathsCount]float32{20, 500, 300, 35},
			best:    [pathsCount]int{0, 0, 0, 0},
		},
		{
			// big differences, public not best
			latency: [pathsCount]float32{500, 1000, 1000, 200},
			best:    [pathsCount]int{3, 3, 3, 3},
		},
		{
			// small diffs, public best
			latency: [pathsCount]float32{100, 105, 110, 120},
			best:    [pathsCount]int{0, 0, 0, 0},
		},
		{
			// small diffs, public not best
			latency: [pathsCount]float32{100, 110, 85, 75},
			best:    [pathsCount]int{3, 3, 3, 3},
		},
		{
			// small diffs, public not best
			latency: [pathsCount]float32{90, 85, 82, 99},
			best:    [pathsCount]int{0, 0, 0, 0},
		},
		{
			// Test with correct thresholds
			latency: [pathsCount]float32{100, 90, 82, 85},
			best:    [pathsCount]int{2, 2, 2, 2},
		},
		{
			// Test with thresholds not reached
			latency: [pathsCount]float32{100, 95, 105, 100},
			best:    [pathsCount]int{0, 0, 0, 0},
		},
	}

	for testIndex, test := range testData {
		rs := New(peerlist.NewPeerList(cfg.AverageSize), &cfg)
		drs := rs.(*directRouteSelector)

		for i, latency := range test.latency {
			drs.fillStats(i, latency, 0)
		}

		for j := 0; j < pathsCount; j++ {
			// invalidate previous best
			drs.bestRoute = netip.Prefix{}

			best := rs.BestPath()
			if best.IP != generateIP(test.best[j]).Addr() {
				t.Errorf("Momentary direct route selector test %d/%d failed (%s vs %s). %s",
					testIndex, j, best.IP, generateIP(test.best[j]), drs.reason.String())
			}
		}
	}
}

// Tests DirectRoute selector taking into account previous values
func TestDirectRouteSelector(t *testing.T) {
	cfg := routeselector.RouteSelectorConfig{
		AverageSize:              10,
		RouteStrategy:            config.RouteStrategyDirectRoute,
		RerouteRatio:             1.1,
		RerouteDiff:              10,
		RouteDeleteLossThreshold: 0,
	}

	testData := []testEntry{
		{
			latency: [pathsCount]float32{20, 500, 300, 35},
			best:    [pathsCount]int{0, 0, 0, 0},
		},
		{
			latency: [pathsCount]float32{500, 1000, 1000, 200},
			best:    [pathsCount]int{3, 3, 3, 3},
		},
		{
			latency: [pathsCount]float32{100, 105, 110, 120},
			best:    [pathsCount]int{0, 0, 0, 0},
		},
		{
			latency: [pathsCount]float32{100, 110, 85, 75},
			best:    [pathsCount]int{3, 3, 3, 3},
		},
		{
			latency: [pathsCount]float32{90, 89, 85, 99},
			best:    [pathsCount]int{0, 0, 2, 0},
		},
		{
			latency: [pathsCount]float32{90, 89, 87, 92},
			best:    [pathsCount]int{0, 0, 0, 0},
		},
		{
			latency: [pathsCount]float32{100, 95, 93, 100},
			best:    [pathsCount]int{0, 1, 2, 0},
		},
		{
			latency: [pathsCount]float32{100, 105, 95, 100},
			best:    [pathsCount]int{0, 0, 2, 0},
		},
	}

	for testIndex, test := range testData {
		rs := New(peerlist.NewPeerList(cfg.AverageSize), &cfg)
		drs := rs.(*directRouteSelector)

		for i, latency := range test.latency {
			drs.fillStats(i, latency, 0)
		}

		for j := 0; j < pathsCount; j++ {
			drs.bestRoute = generateIP(j)

			best := rs.BestPath()
			if best.IP != generateIP(test.best[j]).Addr() {
				t.Errorf("Direct route selector test %d/%d failed (%s vs %s). %s",
					testIndex, j, best.IP, generateIP(test.best[j]), drs.reason.String())
			}
		}
	}
}

func TestDirectRouteThresholdsChange(t *testing.T) {
	cfg := routeselector.RouteSelectorConfig{
		AverageSize:              10,
		RouteStrategy:            config.RouteStrategyDirectRoute,
		RerouteRatio:             1.1,
		RerouteDiff:              10,
		RouteDeleteLossThreshold: 0,
	}

	rs := New(peerlist.NewPeerList(cfg.AverageSize), &cfg)
	drs := rs.(*directRouteSelector)

	drs.fillStats(0, 100, 0)
	drs.fillStats(1, 105, 0)
	drs.fillStats(2, 91, 0)
	drs.fillStats(3, 95, 0)

	best := rs.BestPath()
	if best.IP != generateIP(0).Addr() {
		t.Errorf("Invalid initial best %s", best.IP)
	}

	best = rs.BestPath()
	if best.IP != generateIP(0).Addr() {
		t.Errorf("Invalid initial best retry %s", best.IP)
	}

	drs.addStats(2, 85, 0) // Latency = 90.4
	best = rs.BestPath()
	if best.IP != generateIP(0).Addr() {
		t.Errorf("Invalid not reached thresholds best %s", best.IP)
	}

	drs.addStats(2, 80, 0) // Latency = 89.3
	best = rs.BestPath()
	if best.IP != generateIP(2).Addr() {
		t.Errorf("Invalid reached thresholds best %s", best.IP)
	}

	drs.addStats(3, 10, 0) // Latency = 86.5 vs 89.3 (previous)
	best = rs.BestPath()
	if best.IP != generateIP(2).Addr() {
		t.Errorf("Invalid incremented, but not reached thresholds best %s", best.IP)
	}

}

func TestDirectRouteUnderdog(t *testing.T) {
	cfg := routeselector.RouteSelectorConfig{
		AverageSize:              10,
		RouteStrategy:            config.RouteStrategyDirectRoute,
		RerouteRatio:             1.1,
		RerouteDiff:              10,
		RouteDeleteLossThreshold: 0,
	}

	rs := New(peerlist.NewPeerList(cfg.AverageSize), &cfg)
	drs := rs.(*directRouteSelector)

	// *** Public route as underdog test ***
	// prepare setup as public route is not the best,
	// but is already considered as underdog
	drs.fillStats(0, 90, 0)
	drs.fillStats(1, 89, 0)
	drs.fillStats(2, 85, 0)
	drs.fillStats(3, 99, 0)
	drs.bestRoute = generateIP(2)

	var best *routeselector.SelectedRoute
	// check while public route as underdog
	for i := 0; i <= int(cfg.AverageSize); i++ {
		best = rs.BestPath()
		if best.IP != generateIP(2).Addr() {
			t.Errorf("Invalid best (public underdog not yet ready) %s", best.IP)
		}
	}

	// public underdog should be ready by now
	best = rs.BestPath()
	if best.IP != generateIP(0).Addr() {
		t.Errorf("Invalid best (public route as underdog was expected) %s", best.IP)
	}

	// *** Alt SDN route test
	// prepare setup as one SDN route was chosen before,
	// but alternative SDN is a long term underdog
	drs.fillStats(0, 290, 0)
	drs.fillStats(1, 89, 0)
	drs.fillStats(2, 85, 0)
	drs.fillStats(3, 99, 0)
	drs.bestRoute = generateIP(1)

	// check while alt sdn route as underdog
	for i := 0; i <= int(cfg.AverageSize); i++ {
		best = rs.BestPath()
		if best.IP != generateIP(1).Addr() {
			t.Errorf("Invalid best (public underdog not yet ready) %s", best.IP)
		}
	}

	// sdn underdog should be ready by now
	best = rs.BestPath()
	if best.IP != generateIP(2).Addr() {
		t.Errorf("Invalid best (public route as underdog was expected) %s", best.IP)
	}

}
