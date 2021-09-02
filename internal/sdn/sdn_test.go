package sdn_test

import (
	"testing"

	"github.com/SyntropyNet/syntropy-agent-go/internal/sdn"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
)

func TestSDN(t *testing.T) {
	sdn := &sdn.SdnMonitor{}

	sdn.AddNode("1.1.1.1", "1.1.1.9")
	sdn.AddNode("1.1.1.1", "1.1.1.9") // dupplicate peers should be handled and skipped internally
	sdn.AddNode("2.2.2.1", "2.2.2.9")
	sdn.AddNode("3.3.3.1", "3.3.3.9")
	sdn.AddNode("4.4.4.1", "4.4.4.9")

	peers := sdn.Peers()

	// validate peers count
	if len(peers) != 4 {
		t.Error("Invalid peers count")
	}

	// validate peers
	for _, p := range peers {
		switch p {
		case "1.1.1.9":
			// OK, do nothing
		case "2.2.2.9":
			// OK, do nothing
		case "3.3.3.9":
			// OK, do nothing
		case "4.4.4.9":
			// OK, do nothing
		default:
			t.Errorf("unexpected peer %s", p)
		}
	}

	// simulate ping results
	sdn.PingProcess([]multiping.PingResult{
		{IP: "1.1.1.9", Loss: 0, Latency: 10}, // Medium result
		{IP: "2.2.2.9", Loss: 1, Latency: 0},  // Lowest Latency, but packet Loss
		{IP: "3.3.3.9", Loss: 0, Latency: 3},  // Expected best
		{IP: "4.4.4.9", Loss: 0, Latency: 5},  // Best is not the last
	})

	// NOTE: best gateway is not best peer ;-)
	best := sdn.BestPath()
	if best != "3.3.3.1" {
		t.Errorf("best path calculation problem: %s vs %s (expected)", best, "3.3.3.1")
	}

}
