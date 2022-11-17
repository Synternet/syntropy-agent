package pingdata

import (
	"testing"
	"time"
)

const testRtt = 100 * time.Millisecond
const testSeq = 1111

func TestPingStats(t *testing.T) {
	var s PingStats

	if s.Loss() != 0 || s.Latency() != 0 || s.Duplicate() != 0 {
		t.Fatal("Invalid initial values")
	}

	s.Send(testSeq)
	s.Recv(testSeq, testRtt)
	if s.Loss() != 0 {
		t.Fatal("Ping test failed")
	}
	if s.Duplicate() != 0 {
		t.Fatal("Initial duplicates failed")
	}
	if s.Latency() != float32(testRtt/time.Millisecond) {
		t.Fatalf("Latency test failed %f", s.Latency())
	}

	s.Recv(testSeq, testRtt)
	if s.Duplicate() == 0 {
		t.Fatal("Duplicates test failed")
	}
}
