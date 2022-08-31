package multiping

import (
	"net/netip"
	"sync"
	"time"
)

// Unified interface to process ping data
type PingClient interface {
	PingProcess(pr *PingData)
}

// A single host ping statistics
type PingStats struct {
	tx     uint
	rx     uint
	rtt    time.Duration
	avgRtt time.Duration
}

// Reset statistics to zero values
func (s *PingStats) Reset() {
	s.tx = 0
	s.rx = 0
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
		return float32(s.avgRtt.Microseconds()) / 1000
	} else {
		return 0
	}
}

// Rtt returns last packet rtt
func (s *PingStats) Rtt() time.Duration {
	return s.rtt
}

// Ping data. Holds host information and ping statistics.
// Use Add, Get and Iterate functions. No internal logic will be exposed.
type PingData struct {
	mutex   sync.RWMutex
	entries map[netip.Addr]*PingStats
}

func NewPingData() *PingData {
	return &PingData{
		entries: make(map[netip.Addr]*PingStats),
	}
}

// Add - adds some hosts to be pinged
func (pr *PingData) Add(hosts ...netip.Addr) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	for _, ip := range hosts {
		pr.entries[ip] = &PingStats{}
	}
}

// Del removes some hosts from ping list
func (pr *PingData) Del(hosts ...netip.Addr) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	for _, ip := range hosts {
		delete(pr.entries, ip)
	}
}

// Append - merges 2 PingData into one
func (pr *PingData) Append(data *PingData) {
	pr.mutex.Lock()
	data.mutex.Lock()
	defer pr.mutex.Unlock()
	defer data.mutex.Unlock()

	for ip, stats := range data.entries {
		val, ok := pr.entries[ip]
		if ok {
			if val.rx+stats.rx > 0 {
				val.avgRtt = (val.avgRtt*time.Duration(val.rx) + stats.avgRtt*time.Duration(stats.rx)) /
					time.Duration(val.rx+stats.rx)
			}
			val.rtt = stats.rtt
			val.tx = val.tx + stats.tx
			val.rx = val.rx + stats.rx
		} else {
			pr.entries[ip] = stats
		}
	}
}

// Flush removes all configured hosts
func (pr *PingData) Flush() {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	for h := range pr.entries {
		delete(pr.entries, h)
	}
}

// Reset statistics. Host list remains unchainged.
func (pr *PingData) Reset() {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	for _, e := range pr.entries {
		e.Reset()
	}
}

// Returns count of configured host
func (pr *PingData) Count() int {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	return len(pr.entries)
}

// Get searches for ping statistics of a host
func (pr *PingData) Get(ip netip.Addr) (PingStats, bool) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	val, ok := pr.entries[ip]
	if ok {
		return *val, true
	}

	return PingStats{}, false
}

// Iterate runs through all hosts and calls callback for stats processing
func (pr *PingData) Iterate(callback func(ip netip.Addr, val PingStats)) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	for key, val := range pr.entries {
		callback(key, *val)
	}
}
