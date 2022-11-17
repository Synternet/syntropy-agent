package pingdata

import (
	"fmt"
	"time"
)

// A single host ping statistics
type PingStats struct {
	sequence uint16
	tx       uint
	rx       uint
	dup      uint
	rtt      time.Duration
	avgRtt   time.Duration
}

// Reset statistics to zero values
func (s *PingStats) Reset() {
	s.tx = 0
	s.rx = 0
	s.dup = 0
	s.sequence = 0
	s.rtt = 0
	s.avgRtt = 0
}

func (s *PingStats) Valid() bool {
	return s.tx > 0 && s.tx >= s.rx
}

// Loss returns calculated ping loss
func (s *PingStats) Loss() float32 {
	if s.Valid() {
		return float32(s.tx-s.rx) / float32(s.tx)
	}
	return 0
}

// Latency returns average latency in miliseconds
func (s *PingStats) Latency() float32 {
	if s.Valid() && s.rx > 0 {
		return float32(s.avgRtt / time.Millisecond)
	} else {
		return 0
	}
}

func (s *PingStats) Duplicate() uint {
	return s.dup
}

// Rtt returns last packet rtt
func (s *PingStats) Rtt() time.Duration {
	return s.rtt
}

func (s *PingStats) String() string {
	return fmt.Sprintf("tx=%d, rx=%d, rtt=%s, avgRtt=%s",
		s.tx, s.rx, s.rtt, s.avgRtt)
}

func (s *PingStats) Send(seq uint16) {
	s.tx++
	s.rtt = 0
	s.sequence = seq
}

func (s *PingStats) Recv(seq uint16, rtt time.Duration) {
	if s.sequence == seq {
		s.rx++
		s.rtt = rtt
		if s.avgRtt == 0 {
			s.avgRtt = rtt
		} else {
			s.avgRtt = (time.Duration(s.rx)*s.avgRtt + s.rtt) / time.Duration(s.rx+1)
		}
		s.sequence = 0
	} else {
		s.dup++
	}
}
