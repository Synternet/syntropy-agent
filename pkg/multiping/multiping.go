package multiping

import (
	"sync"
	"time"

	"github.com/go-ping/ping"
)

type MultiPing struct {
	Timeout     time.Duration
	PacketCount int
	pingClient  PingClient
	hosts       PingResult
}

func New(p PingClient) *MultiPing {
	mp := &MultiPing{
		pingClient:  p,
		PacketCount: 1,
		Timeout:     time.Second,
		hosts: PingResult{
			data: make(map[string]*PingResultValue),
		},
	}

	return mp
}

func (p *MultiPing) AddHost(hosts ...string) {
	p.hosts.mutex.Lock()
	defer p.hosts.mutex.Unlock()

	for _, ip := range hosts {
		p.hosts.data[ip] = &PingResultValue{}
	}
}

func (p *MultiPing) DelHost(hosts ...string) {
	p.hosts.mutex.Lock()
	defer p.hosts.mutex.Unlock()

	for _, ip := range hosts {
		delete(p.hosts.data, ip)
	}
}

// Remove all configured hosts
func (p *MultiPing) Flush() {
	p.hosts.mutex.Lock()
	defer p.hosts.mutex.Unlock()

	for ip, _ := range p.hosts.data {
		delete(p.hosts.data, ip)
	}
}

// Pings a host and fills results
func (p *MultiPing) pingHost(wgroup *sync.WaitGroup, host string) {
	defer wgroup.Done()

	pinger, err := ping.NewPinger(host)
	if err != nil {
		return
	}
	pinger.SetPrivileged(true)
	pinger.Count = p.PacketCount
	pinger.Timeout = p.Timeout

	err = pinger.Run()
	if err != nil {
		return
	}

	p.hosts.update(host, pinger.Statistics())
}

// Pings configured hosts and calls an instance of PingClient with collected results.
func (p *MultiPing) Ping() {
	count := p.hosts.Count()

	wg := sync.WaitGroup{}
	wg.Add(count)

	// Spawn all host pinging to goroutines
	p.hosts.mutex.RLock()
	for key, _ := range p.hosts.data {
		go p.pingHost(&wg, key)
	}
	p.hosts.mutex.RUnlock()

	// Wait for the results and process them
	wg.Wait()
	p.pingClient.PingProcess(&p.hosts)
}

func (p *MultiPing) Count() int {
	return p.hosts.Count()
}
