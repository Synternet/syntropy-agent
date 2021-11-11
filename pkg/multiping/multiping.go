package multiping

//go:generate mockgen -source=multiping.go -destination=mock.go -package=multiping

import (
	"context"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/pkg/scontext"
	"github.com/go-ping/ping"
)

type PingResult struct {
	IP      string  `json:"ip"`
	Latency float32 `json:"latency_ms,omitempty"`
	Loss    float32 `json:"packet_loss"`
}

type PingClient interface {
	PingProcess(pr []PingResult)
}

type Pinger interface {
	Run() error
	Statistics() *ping.Statistics
}

type MultiPing struct {
	sync.RWMutex
	ctx        scontext.StartStopContext
	Timeout    time.Duration
	Period     time.Duration
	pinger     func(string) (Pinger, error)
	pingClient PingClient
	hosts      []string
	Count      int
	LimitCount int
}

func New(ctx context.Context, p PingClient) *MultiPing {
	mp := &MultiPing{
		pingClient: p,
		Period:     0,
		Count:      1,
		Timeout:    1 * time.Second,
		LimitCount: 1000,
		ctx:        scontext.New(ctx),
	}
	mp.pinger = mp.pingerCreator
	return mp
}

// This abstraction is required for testing
// Otherwise the interface of this package should be changed to support
// dependency injection pattern.
// However, sadly, ping library used here, does not expose proper API that is suitable for testing.
// Hence the wrapper.
func (p *MultiPing) pingerCreator(addr string) (Pinger, error) {
	pinger, err := ping.NewPinger(addr)
	if err != nil {
		return nil, err
	}
	pinger.SetPrivileged(true)
	pinger.Count = p.Count
	pinger.Timeout = p.Timeout
	return pinger, err
}

func (p *MultiPing) AddHost(hosts ...string) {
	p.Lock()
	defer p.Unlock()

	for _, h := range hosts {
		dupplicate := false
		for _, e := range p.hosts {
			if e == h {
				dupplicate = true
				break
			}
		}
		if !dupplicate {
			p.hosts = append(p.hosts, h)
		}
	}
}

func (p *MultiPing) DelHost(hosts ...string) {
	p.Lock()
	defer p.Unlock()

	for _, h := range hosts {
		for i, e := range p.hosts {
			if e == h {
				// remove host, if found
				// order is not important, so I'm trying to reduce reallocations
				p.hosts[i] = p.hosts[len(p.hosts)-1]
				p.hosts = p.hosts[:len(p.hosts)-1]
				break
			}
		}
	}
}

// Remove all configured hosts
func (p *MultiPing) Flush() {
	p.Lock()
	defer p.Unlock()
	p.hosts = []string{}
}

// Pings a host given host index
// Fills in PingResult slice. Since concurrent hostIndex are unique, there is no collision.
func (p *MultiPing) pingHost(wgroup *sync.WaitGroup, hostIndex int, results []PingResult) {
	defer wgroup.Done()

	host := p.hosts[hostIndex]
	results[hostIndex] = PingResult{
		IP:      host,
		Latency: 0,
		Loss:    1,
	}

	pinger, err := p.pinger(host)
	if err != nil {
		return
	}

	err = pinger.Run()
	if err != nil {
		return
	}

	stats := pinger.Statistics()
	results[hostIndex].IP = stats.Addr
	results[hostIndex].Loss = float32(stats.PacketLoss) / 100
	if stats.PacketLoss >= 100 {
		results[hostIndex].Latency = 0
	} else {
		results[hostIndex].Latency = float32(stats.AvgRtt.Microseconds()) / 1000
	}
}

// Pings configured hosts and calls an instance of PingClient with collected results.
func (p *MultiPing) Ping() {
	p.RLock()
	defer p.RUnlock()

	count := len(p.hosts)
	results := make([]PingResult, count)
	wg := sync.WaitGroup{}
	wg.Add(count)

	// Ping results listener. Waits for all the entries in results to be
	// filled concurrently. Sends the results for processing.
	go func() {
		wg.Wait()
		p.pingClient.PingProcess(results)
	}()

	// Spawn all host pinging to goroutines
	for i := range p.hosts {
		// In this case sharing memory is more efficient and readable
		go p.pingHost(&wg, i, results)
	}
}

// Starts configured pinger
func (p *MultiPing) Start() {
	p.Lock()
	defer p.Unlock()
	if p.Period == 0 {
		return
	}

	ctx, _ := p.ctx.CreateContext()
	go func() {
		ticket := time.NewTicker(p.Period)
		defer ticket.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticket.C:
				p.Ping()
			}
		}
	}()
}

// Stop stops running pinger and removes all configured hosts
func (p *MultiPing) Stop() {
	p.Lock()
	defer p.Unlock()

	// Stop the goroutine
	p.ctx.CancelContext()
}
