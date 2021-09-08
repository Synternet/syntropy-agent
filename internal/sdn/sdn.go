package sdn

import (
	"fmt"
	"sync"

	"github.com/SyntropyNet/syntropy-agent-go/pkg/multiping"
)

const pkgName = "PeerMonitor. "

type peerInfo struct {
	endpoint string
	gateway  string
	latency  int
	loss     float32
}

func (node *peerInfo) String() string {
	return fmt.Sprintf("%s via %s loss: %f latency %d", node.endpoint, node.gateway, node.loss, node.latency)
}

type SdnMonitor struct {
	sync.RWMutex
	list []*peerInfo
}

func (sdn *SdnMonitor) AddNode(gw, peer string) {
	sdn.Lock()
	defer sdn.Unlock()

	for _, n := range sdn.list {
		if n.endpoint == peer {
			return
		}
	}

	e := peerInfo{
		gateway:  gw,
		endpoint: peer,
	}
	sdn.list = append(sdn.list, &e)
}

func (sdn *SdnMonitor) Peers() []string {
	sdn.RLock()
	defer sdn.RUnlock()

	rv := []string{}

	for _, e := range sdn.list {
		rv = append(rv, e.endpoint)
	}
	return rv
}

func (sdn *SdnMonitor) PingProcess(pr []multiping.PingResult) {
	sdn.Lock()
	defer sdn.Unlock()

	for _, res := range pr {
		for _, peer := range sdn.list {
			if peer.endpoint == res.IP {
				peer.latency = res.Latency
				peer.loss = res.Loss
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
	for i := bestIdx + 1; i < len(sdn.list); i++ {
		if sdn.list[i].loss < sdn.list[bestIdx].loss {
			bestIdx = i
		} else if sdn.list[i].latency > 0 && sdn.list[i].latency < sdn.list[bestIdx].latency {
			bestIdx = i
		}
	}

	return sdn.list[bestIdx].gateway
}
