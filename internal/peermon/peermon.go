package peermon

// TODO: think about merging this with router.Router and using some interfaces

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

type PeerMonitor struct {
	sync.RWMutex
	list []*peerInfo
}

func (pm *PeerMonitor) AddNode(gw, peer string) {
	pm.Lock()
	defer pm.Unlock()

	for _, n := range pm.list {
		if n.endpoint == peer {
			return
		}
	}

	e := peerInfo{
		gateway:  gw,
		endpoint: peer,
	}
	pm.list = append(pm.list, &e)
}

func (pm *PeerMonitor) Peers() []string {
	pm.RLock()
	defer pm.RUnlock()

	rv := []string{}

	for _, e := range pm.list {
		rv = append(rv, e.endpoint)
	}
	return rv
}

func (pm *PeerMonitor) PingProcess(pr []multiping.PingResult) {
	pm.Lock()
	defer pm.Unlock()

	for _, res := range pr {
		for _, peer := range pm.list {
			if peer.endpoint == res.IP {
				peer.latency = res.Latency
				peer.loss = res.Loss
				break // break internal loop, continue on external
			}
		}
	}

}

func (pm *PeerMonitor) BestPath() string {
	pm.RLock()
	defer pm.RUnlock()

	if len(pm.list) == 0 {
		return ""
	}

	bestIdx := 0
	for i := bestIdx + 1; i < len(pm.list); i++ {
		if pm.list[i].loss < pm.list[bestIdx].loss {
			bestIdx = i
		} else if pm.list[i].latency > 0 && pm.list[i].latency < pm.list[bestIdx].latency {
			bestIdx = i
		}
	}

	return pm.list[bestIdx].gateway
}
