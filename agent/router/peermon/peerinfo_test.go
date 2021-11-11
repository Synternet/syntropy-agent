package peermon

import "testing"

func TestPeerInfo(t *testing.T) {
	pi := peerInfo{}

	// test incomplete results
	for i := 0; i < 3; i++ {
		pi.Add(float32(i), 0)
	}
	if pi.Loss() != 0 {
		t.Errorf("invalid loss (0 expected)")
	}
	if pi.Latency() != 1.5 {
		t.Errorf("invalid loss (1.5 expected)")
	}

	// test with overwrapped results
	for i := 3; i < 20; i++ {
		if i == 15 {
			pi.Add(float32(i), 2)
		} else {
			pi.Add(float32(i), 0)
		}
	}

	if pi.Loss() != 0.2 {
		t.Errorf("invalid loss (0.2 expected)")
	}
	if pi.Latency() != 14.5 {
		t.Errorf("invalid loss (14.5 expected)")
	}
}
