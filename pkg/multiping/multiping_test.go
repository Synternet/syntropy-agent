package multiping

import (
	"fmt"
	"testing"
)

func TestMultiping(t *testing.T) {
	const maxCount = 222
	data := NewPingData()
	// Sad truth - agent uses privileged pinger, but in that case tests require root
	pinger, err := New(false)

	if err != nil {
		t.Errorf("Multiping constructor failed %s", err)
	}

	var invalidHost = "some.invalid.host"
	for i := 1; i <= maxCount; i++ {
		data.Add(fmt.Sprintf("127.0.0.%d", i))
	}
	data.Add(invalidHost)
	pinger.Ping(data)
	if data.Count() != maxCount {
		t.Errorf("Pinger accepts invald IP address")
	}

	val, ok := data.Get("127.0.0.1")
	if !ok {
		t.Errorf("Expected localhost missing")
	}
	if val.Loss() != 0 {
		t.Errorf("Localhost ping failed")
	}
	if val.Latency() == 0 {
		t.Errorf("Localhost invalid latency")
	}

	val, ok = data.Get(invalidHost)
	if ok {
		t.Errorf("Pinger has invalid host")
	}
	if val.Loss() != 0 {
		t.Errorf("Non existing host invalid loss")
	}
	if val.Latency() != 0 {
		t.Errorf("Non existing host invalid latency")
	}

	val, ok = data.Get("no such host")
	if ok {
		t.Errorf("Non existing host search failed")
	}
	if val.Loss() != 0 {
		t.Errorf("Non existing host invalid loss")
	}
	if val.Latency() != 0 {
		t.Errorf("Non existing host invalid latency")
	}

}
