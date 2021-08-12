package pinger

import (
	"sync"
	"time"

	"github.com/go-ping/ping"
)

type PingResult struct {
	IP      string  `json:"ip"`
	Latency int     `json:"latency_ms"`
	Loss    float32 `json:"packet_loss"`
}

type PingResultProcessor interface {
	ProcessPingResults(pr []PingResult)
}

type Pinger struct {
	sync.RWMutex
	stop       chan bool
	running    bool
	prp        PingResultProcessor
	hosts      []string
	period     time.Duration
	limitCount int
}

func NewPinger(p PingResultProcessor) *Pinger {
	return &Pinger{
		prp:        p,
		period:     time.Minute,
		limitCount: 1000,
		stop:       make(chan bool),
		running:    false,
	}
}

func (p *Pinger) Setup(period time.Duration, limit int) {
	p.Lock()
	defer p.Unlock()
	p.period = period
	p.limitCount = limit
}

func (p *Pinger) AddHost(hosts ...string) {
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

func (p *Pinger) DelHost(hosts ...string) {
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

func pingHost(h string, c chan PingResult) {
	pinger, err := ping.NewPinger(h)
	res := PingResult{
		IP:      h,
		Latency: -1,
		Loss:    1,
	}
	defer func() { c <- res }()

	if err != nil {
		return
	}

	pinger.Count = 2
	pinger.Timeout = time.Second

	err = pinger.Run()
	if err != nil {
		return
	}

	stats := pinger.Statistics()
	res.IP = stats.Addr
	res.Loss = float32(stats.PacketLoss) / 100
	if res.Loss == 1 {
		res.Latency = -1
	} else {
		res.Latency = int(stats.AvgRtt.Milliseconds())
	}
	// `res` added to channel from defer
}

func (p *Pinger) pingAll() {
	p.RLock()
	defer p.RUnlock()

	c := make(chan PingResult)
	count := len(p.hosts)

	// Ping results listener. Reads count of hosts entries from channel
	// Closes the channel and sends collected results
	go func() {
		var result []PingResult
		for count > 0 {
			r := <-c
			result = append(result, r)
			count--
		}
		close(c)
		p.prp.ProcessPingResults(result)
	}()

	// Spawn all host pinging to goroutines
	for i := 0; i < len(p.hosts); i++ {
		go pingHost(p.hosts[i], c)
	}
}

// Starts configured pinger
func (p *Pinger) Start() {
	p.Lock()
	defer p.Unlock()
	if p.running {
		// Do not start new pinger, if one is alredy running
		return
	}
	p.running = true

	t := time.NewTicker(p.period)

	go func() {
		for {
			select {
			case <-p.stop:
				return
			case <-t.C:
				p.pingAll()

			}
		}
	}()
}

// Stop stops running pinger and removes all configured hosts
func (p *Pinger) Stop() {
	p.Lock()
	defer p.Unlock()

	if p.running {
		p.running = false

		// Stop the goroutine
		p.stop <- true
	}

	// remove all configured hosts
	p.hosts = []string{}
}

// Runs the configured pinger only once
func (p *Pinger) RunOnce() {
	p.Lock()
	defer p.Unlock()

	p.pingAll()
}
