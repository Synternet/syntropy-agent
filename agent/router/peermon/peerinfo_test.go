package peermon

import "testing"

func TestPeerInfo(t *testing.T) {
	valuesCount := 20
	pi := newPeerInfo(uint(valuesCount))

	// Test if empty results are valid
	if pi.Valid() {
		t.Errorf("invalid results (false expected)")
	}

	// test incomplete results
	for i := 0; i < 3; i++ {
		pi.Add(float32(i), 0)
	}
	if !pi.Valid() {
		t.Errorf("invalid results (true expected)")
	}
	if pi.Loss() != 0 {
		t.Errorf("invalid loss (0 expected)")
	}
	if pi.Latency() != 1.5 {
		t.Errorf("invalid loss (1.5 expected)")
	}

	// test with overwrapped results
	sumVal := 50
	for i := 3; i <= sumVal; i++ {
		if i == sumVal-3 {
			pi.Add(float32(i), float32(2))
		} else {
			pi.Add(float32(i), 0)
		}
	}

	expectedLoss := float32(2) / float32(valuesCount)
	expectedLatency := float32(0)
	for i := 0; i < valuesCount; i++ {
		expectedLatency = expectedLatency + float32(sumVal)
		sumVal--
	}
	expectedLatency = expectedLatency / float32(valuesCount)

	if pi.Loss() != expectedLoss {
		t.Errorf("invalid loss %f (%f expected)", pi.Loss(), expectedLoss)
	}
	if pi.Latency() != expectedLatency {
		t.Errorf("invalid latency %f (%f expected)", pi.Latency(), expectedLatency)
	}
}
