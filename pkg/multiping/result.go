package multiping

import (
	"sync"

	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping/ping"
)

type PingClient interface {
	PingProcess(pr *PingResult)
}

type PingResultValue struct {
	Latency float32
	Loss    float32
}

type PingResult struct {
	mutex sync.RWMutex
	data  map[string]*PingResultValue
}

func (pr *PingResult) Count() int {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	return len(pr.data)
}

func (pr *PingResult) Get(ip string) (PingResultValue, bool) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	val, ok := pr.data[ip]
	if ok {
		return *val, true
	}

	return PingResultValue{}, false
}

func (pr *PingResult) Iterate(callback func(ip string, val PingResultValue)) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	for key, val := range pr.data {
		callback(key, *val)
	}
}

func (pr *PingResult) update(ip string, stats *ping.Statistics) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	val, ok := pr.data[ip]
	if !ok {
		return
	}

	if stats.PacketLoss >= 100 {
		val.Latency = 0
	} else {
		val.Latency = float32(stats.AvgRtt.Microseconds()) / 1000
	}
	val.Loss = float32(stats.PacketLoss) / 100
}
