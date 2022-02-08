package peermon

import (
	"fmt"
)

// peerInfo collects stores and calculates moving average of last [SYNTROPY_PEERCHECK_WINDOW] link measurement
type peerInfo struct {
	ifname       string
	publicKey    string
	connectionID int
	endpoint     string
	gateway      string
	latency      []float32
	loss         []float32
	index        int
}

func newPeerInfo(avgCount uint) *peerInfo {
	pi := peerInfo{
		latency: make([]float32, avgCount),
		loss:    make([]float32, avgCount),
	}
	return &pi
}

func (node *peerInfo) Add(latency, loss float32) {
	node.latency[node.index] = latency
	node.loss[node.index] = loss
	node.index++
	if node.index >= cap(node.latency) {
		node.index = 0
	}
}

func (node *peerInfo) Latency() float32 {
	count := 0
	var sum float32
	for _, val := range node.latency {
		if val > 0 {
			sum = sum + val
			count++
		}
	}
	if count > 0 {
		return sum / float32(count)
	}
	return 0
}

func (node *peerInfo) Loss() float32 {
	count := 0
	var sum float32
	for _, val := range node.loss {
		sum = sum + val
		count++
	}
	if count > 0 {
		return sum / float32(count)
	}
	return 0
}

func (node *peerInfo) StatsIncomplete() bool {
	count := 0
	for _, val := range node.latency {
		if val > 0 {
			count++
		}
	}
	return count != cap(node.latency)
}

func (node *peerInfo) String() string {
	return fmt.Sprintf("%s dev %s loss: %f latency %f",
		node.endpoint, node.ifname, node.Loss(), node.Latency())
}
