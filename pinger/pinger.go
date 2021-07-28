package pinger

import (
	"io"
	"log"
	"sync"
	"time"
)

type Pinger struct {
	sync.RWMutex
	stop       chan bool
	running    bool
	w          io.Writer
	hosts      []string
	period     time.Duration
	limitCount int
}

func NewPinger(w io.Writer) *Pinger {
	return &Pinger{
		w:          w,
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

func pingHost(h string) {
	log.Println("Pinging ", h)
}

func (p *Pinger) pingAll() {
	p.RLock()
	defer p.RUnlock()

	for i := 0; i < len(p.hosts); i++ {
		pingHost(p.hosts[i])

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

	if !p.running {
		return
	}

	p.running = false

	// Stop the goroutine
	p.stop <- true

	// remove all configured hosts
	p.hosts = []string{}
}
