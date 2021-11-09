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
	latency  float32
	loss     float32
}

func (node *peerInfo) String() string {
	return fmt.Sprintf("%s via %s loss: %f latency %f", node.endpoint, node.gateway, node.loss, node.latency)
}

type PeerMonitor struct {
	sync.RWMutex
	peerList []*peerInfo
}

func (pm *PeerMonitor) AddNode(gateway, endpoint string) {
	pm.Lock()
	defer pm.Unlock()

	for _, peer := range pm.peerList {
		if peer.endpoint == endpoint {
			return
		}
	}

	e := peerInfo{
		gateway:  gateway,
		endpoint: endpoint,
	}
	pm.peerList = append(pm.peerList, &e)
}

func (pm *PeerMonitor) DelNode(endpoint string) {
	pm.Lock()
	defer pm.Unlock()

	for idx, peer := range pm.peerList {
		if peer.endpoint == endpoint {
			// order is not important.
			// Remove from slice in more effective way
			pm.peerList[idx] = pm.peerList[len(pm.peerList)-1]
			pm.peerList = pm.peerList[:len(pm.peerList)-1]
			return
		}
	}
}

func (pm *PeerMonitor) Peers() []string {
	pm.RLock()
	defer pm.RUnlock()

	rv := []string{}

	for _, peer := range pm.peerList {
		rv = append(rv, peer.endpoint)
	}
	return rv
}

func (pm *PeerMonitor) PingProcess(pr []multiping.PingResult) {
	pm.Lock()
	defer pm.Unlock()

	for _, res := range pr {
		for _, peer := range pm.peerList {
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

	if len(pm.peerList) == 0 {
		return ""
	}

	bestIdx := 0
	for i := bestIdx + 1; i < len(pm.peerList); i++ {
		if pm.peerList[i].loss < pm.peerList[bestIdx].loss {
			bestIdx = i
		} else if pm.peerList[i].latency > 0 && pm.peerList[i].latency < pm.peerList[bestIdx].latency {
			bestIdx = i
		}
	}

	return pm.peerList[bestIdx].gateway
}
