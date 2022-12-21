package speed

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

func (srs *speedRouteSelector) fillStats(index int, latency, loss float32) {
	var ifname string
	if index == 0 {
		ifname = "SYNTROPY_PUBLIC"
	} else {
		ifname = fmt.Sprintf("SYNTROPY_SDN%d", index)
	}

	ip := generateIP(index)
	srs.peerlist.AddPeer(ifname, "PublicKey", ip, index, false)

	peer, ok := srs.peerlist.GetPeer(ip)
	if ok {
		peer.ResetFlags()
		for i := 0; i < int(srs.config.AverageSize); i++ {
			peer.Add(latency, loss)
		}
	}
}

func TestRouteSelectorSpeed(t *testing.T) {
	cfg := routeselector.RouteSelectorConfig{
		AverageSize:              10,
		RouteStrategy:            config.RouteStrategySpeed,
		RerouteRatio:             1.1,
		RerouteDiff:              20,
		RouteDeleteLossThreshold: 0,
	}

	testData := []testEntry{
		{
			latency: [pathsCount]float32{20, 500, 300, 35},
			best:    [pathsCount]int{0, 0, 0, 3},
		},
		{
			latency: [pathsCount]float32{500, 1000, 1000, 200},
			best:    [pathsCount]int{3, 3, 3, 3},
		},
		{
			latency: [pathsCount]float32{100, 105, 110, 120},
			best:    [pathsCount]int{0, 1, 2, 0},
		},
		{
			latency: [pathsCount]float32{100, 110, 85, 78},
			best:    [pathsCount]int{3, 3, 2, 3},
		},
		{
			latency: [pathsCount]float32{100, 90, 85, 82},
			best:    [pathsCount]int{0, 1, 2, 3},
		},
		{
			latency: [pathsCount]float32{100, 105, 95, 100},
			best:    [pathsCount]int{0, 1, 2, 3},
		},
	}

	for testIndex, test := range testData {
		rs := New(peerlist.NewPeerList(cfg.AverageSize), &cfg)
		srs := rs.(*speedRouteSelector)

		for i, latency := range test.latency {
			srs.fillStats(i, latency, 0)
		}

		for j := 0; j < pathsCount; j++ {
			srs.bestRoute = generateIP(j)

			best := rs.BestPath()
			if best.IP != generateIP(test.best[j]).Addr() {
				t.Errorf("Speed route selector test %d/%d failed (%s vs %s)",
					testIndex, j, best.IP, generateIP(test.best[j]))
			}
		}
	}

}
