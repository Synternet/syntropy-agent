package sdn

import (
	"sync"

	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
)

type SdnNode struct {
	Endpoint string
	Gateway  string
	Latency  int
	Loss     float32
}

type SdnMonitor struct {
	sync.RWMutex
	list []*SdnNode
}

func (sdn *SdnMonitor) AddNode(gw, peer string) {
	sdn.Lock()
	defer sdn.Unlock()

	for _, n := range sdn.list {
		if n.Endpoint == peer {
			return
		}
	}

	e := SdnNode{
		Gateway:  gw,
		Endpoint: peer,
	}
	sdn.list = append(sdn.list, &e)
}

func (sdn *SdnMonitor) Peers() []string {
	sdn.RLock()
	defer sdn.RUnlock()

	rv := []string{}

	for _, e := range sdn.list {
		rv = append(rv, e.Endpoint)
	}
	return rv
}

func (sdn *SdnMonitor) PingProcess(pr []multiping.PingResult) {
	sdn.Lock()
	defer sdn.Unlock()

	for _, res := range pr {
		for _, peer := range sdn.list {
			if peer.Endpoint == res.IP {
				peer.Latency = res.Latency
				peer.Loss = res.Loss
				break // break internal loop, continue on external
			}
		}
	}

}

func (sdn *SdnMonitor) BestPath() string {
	sdn.RLock()
	defer sdn.RUnlock()

	if len(sdn.list) == 0 {
		return ""
	}

	bestIdx := 0
	for i := bestIdx; i < len(sdn.list); i++ {
		if sdn.list[i].Loss < sdn.list[bestIdx].Loss {
			bestIdx = i
		} else if sdn.list[i].Latency < sdn.list[bestIdx].Latency {
			bestIdx = i
		}
	}

	return sdn.list[bestIdx].Gateway
}
