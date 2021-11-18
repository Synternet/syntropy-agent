package multiping

import (
	"log"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping/ping"
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

// Pings configured hosts and calls an instance of PingClient with collected results.
func (p *MultiPing) Ping() {
	pinger, err := ping.New(true)
	if err != nil {
		log.Println("Pinger error", err)
		return
	}
	pinger.Timeout = time.Second

	hosts := []string{}

	for h, _ := range p.hosts.data {
		hosts = append(hosts, h)
	}

	pinger.Run(hosts)

	stats := pinger.Statistics()

	for ip, stat := range stats {
		p.hosts.update(ip, stat)
	}
	p.pingClient.PingProcess(&p.hosts)
}

func (p *MultiPing) Count() int {
	return p.hosts.Count()
}
