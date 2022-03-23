package twamp

import (
	"time"
)

// A single host twamp ping statistics
// TODO: this is drop-in statistics from multiping package
// Think about merging or an interface in future
type TwampStats struct {
	tx     uint
	rx     uint
	rtt    time.Duration
	avgRtt time.Duration
}

// Reset statistics to zero values
func (s *TwampStats) Reset() {
	s.tx = 0
	s.rx = 0
	s.rtt = 0
}

// Loss returns calculated ping loss
func (s *TwampStats) Loss() float32 {
	if s.tx > 0 {
		return float32(s.tx-s.rx) / float32(s.tx)
	}
	return 0
}

// Latency returns average latency in miliseconds
func (s *TwampStats) Latency() float32 {
	return float32(s.avgRtt.Microseconds()) / 1000
}

// Rtt returns last packet rtt
func (s *TwampStats) Rtt() time.Duration {
	return s.rtt
}

// Tx returns transmitted packets count
func (s *TwampStats) Tx() uint {
	return s.tx
}

// Rx returns received packets count
func (s *TwampStats) Rx() uint {
	return s.rx
}