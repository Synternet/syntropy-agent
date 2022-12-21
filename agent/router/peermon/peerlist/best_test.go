package peerlist

import (
	"fmt"
	"net/netip"
	"testing"
)

const pathsCount = 4
const averageSize = 10

type testEntry struct {
	latency [pathsCount]float32
	best    int
}

func generateIP(i int) netip.Prefix {
	ip := netip.MustParseAddr(fmt.Sprintf("1.1.1.%d", i+1))
	return netip.PrefixFrom(ip, ip.BitLen())
}

func (pl *PeerList) fillStats(index int, latency, loss float32) {
	var ifname string
	if index == 0 {
		ifname = "SYNTROPY_PUBLIC"
	} else {
		ifname = fmt.Sprintf("SYNTROPY_SDN%d", index)
	}

	ip := generateIP(index)
	pl.AddPeer(ifname, "PublicKey", ip, index, false)

	peer, ok := pl.GetPeer(ip)
	if ok {
		peer.flags = PifNone
		for i := 0; i < averageSize; i++ {
			peer.Add(latency, loss)
		}
	}
}

// This tests the best function without previous values
func TestBestFunctions1(t *testing.T) {
	testData := []testEntry{
		{
			latency: [4]float32{20, 500, 300, 35},
			best:    0,
		},
		{
			latency: [4]float32{500, 1000, 1000, 200},
			best:    3,
		},
		{
			latency: [4]float32{100, 105, 110, 120},
			best:    0,
		},
		{
			latency: [4]float32{100, 110, 85, 78},
			best:    3,
		},
		{
			latency: [4]float32{100, 90, 85, 82},
			best:    3,
		},
	}

	for testIndex, test := range testData {
		peerlist := NewPeerList(averageSize)

		for i, latency := range test.latency {
			peerlist.fillStats(i, latency, 0)
		}

		best := peerlist.BestRoute()
		if best != generateIP(test.best) {
			t.Errorf("Lowest latency test %d failed (%s vs %s)", testIndex, best, generateIP(test.best))
		}
	}

}
