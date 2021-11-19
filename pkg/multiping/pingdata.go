package multiping

import (
	"sync"
	"time"
)

// Unified interface to process ping data
type PingClient interface {
	PingProcess(pr *PingData)
}

// A single host ping statistics
type PingStats struct {
	tx  uint
	rx  uint
	rtt time.Duration
}

// Reset statistics to zero values
func (s *PingStats) Reset() {
	s.tx = 0
	s.rx = 0
	s.rtt = 0
}

// Loss returns calculated ping loss
func (s *PingStats) Loss() float32 {
	if s.tx > 0 {
		return float32(s.tx-s.rx) / float32(s.tx)
	}
	return 0
}

// Loss returns latency in miliseconds calculated ping loss
func (s *PingStats) Latency() float32 {
	return float32(s.rtt.Microseconds()) / 1000
}

// Ping data. Holds host information and ping statistics.
// Use Add, Get and Iterate functions. No internal logic will be exposed.
type PingData struct {
	mutex   sync.RWMutex
	entries map[string]*PingStats
}

func NewPingData() *PingData {
	return &PingData{
		entries: make(map[string]*PingStats),
	}
}

// Add - adds some hosts to be pinged
func (pr *PingData) Add(hosts ...string) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	for _, ip := range hosts {
		pr.entries[ip] = &PingStats{}
	}
}

// Del removes some hosts from ping list
func (pr *PingData) Del(hosts ...string) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	for _, ip := range hosts {
		delete(pr.entries, ip)
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
func (pr *PingData) Get(ip string) (PingStats, bool) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	val, ok := pr.entries[ip]
	if ok {
		return *val, true
	}

	return PingStats{}, false
}

// Iterate runs through all hosts and calls callback for stats processing
func (pr *PingData) Iterate(callback func(ip string, val PingStats)) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	for key, val := range pr.entries {
		callback(key, *val)
	}
}
