package pingdata

import (
	"fmt"
	"io"
	"net/netip"
	"time"
)

// Ping data. Holds host information and ping statistics.
// Use Add, Get and Iterate functions. No internal logic will be exposed.
type PingData struct {
	entries map[netip.Addr]*PingStats
}

func NewPingData() *PingData {
	return &PingData{
		entries: make(map[netip.Addr]*PingStats),
	}
}

// Add - adds some hosts to be pinged
func (pr *PingData) Add(hosts ...netip.Addr) {
	for _, ip := range hosts {
		pr.entries[ip] = &PingStats{}
	}
}

// Del removes some hosts from ping list
func (pr *PingData) Del(hosts ...netip.Addr) {
	for _, ip := range hosts {
		delete(pr.entries, ip)
	}
}

// Append - merges 2 PingData into one
func (pr *PingData) Append(data *PingData) {
	data.Iterate(func(ip netip.Addr, stats *PingStats) {
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
	})
}

// Flush removes all configured hosts
func (pr *PingData) Flush() {
	for h := range pr.entries {
		delete(pr.entries, h)
	}
}

// Reset statistics. Host list remains unchainged.
func (pr *PingData) Reset() {
	for _, e := range pr.entries {
		e.Reset()
	}
}

// Returns count of configured host
func (pr *PingData) Count() int {
	return len(pr.entries)
}

// Get searches for ping statistics of a host
func (pr *PingData) Get(ip netip.Addr) (*PingStats, bool) {
	val, ok := pr.entries[ip]
	if ok {
		return val, true
	}

	return nil, false
}

// Iterate runs through all hosts and calls callback for stats processing
func (pr *PingData) Iterate(callback func(ip netip.Addr, val *PingStats)) {
	for key, val := range pr.entries {
		callback(key, val)
	}
}

func (pr *PingData) Dump(w io.Writer, title ...string) {
	for _, l := range title {
		w.Write([]byte(l))
	}

	pr.Iterate(func(ip netip.Addr, val *PingStats) {
		line := fmt.Sprintf("%s: %s\n", ip, val.String())
		w.Write([]byte(line))
	})
}
