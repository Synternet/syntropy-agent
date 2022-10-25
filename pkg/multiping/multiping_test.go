package multiping

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/SyntropyNet/syntropy-agent/pkg/multiping/pingdata"
)

func TestMultiping(t *testing.T) {
	const maxCount = 222
	data := pingdata.NewPingData()
	// Sad truth - agent uses privileged pinger, but in that case tests require root
	pinger, err := New(false)

	if err != nil {
		t.Errorf("Multiping constructor failed %s", err)
	}

	for i := 1; i <= maxCount; i++ {
		data.Add(netip.MustParseAddr(fmt.Sprintf("127.0.0.%d", i)))
	}
	pinger.Ping(data)
	if data.Count() != maxCount {
		t.Errorf("Pinger accepts invald IP address")
	}

	val, ok := data.Get(netip.MustParseAddr("127.0.0.1"))
	if !ok {
		t.Errorf("Expected localhost missing")
	}
	if val.Loss() != 0 {
		t.Errorf("Localhost ping failed: %f", val.Loss())
	}
	if val.Latency() == 0 {
		t.Errorf("Localhost invalid latency %f", val.Latency())
	}

	val, ok = data.Get(netip.IPv4Unspecified())
	if ok {
		t.Errorf("Pinger has invalid host")
	}
	if val != nil {
		t.Errorf("Non existing host invalid stats")
	}
}
