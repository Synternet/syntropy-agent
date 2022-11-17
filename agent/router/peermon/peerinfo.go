package peermon

import (
	"fmt"
	"strings"

	"github.com/SyntropyNet/syntropy-agent/internal/env"
)

const (
	pifNone       = uint8(0x00)
	pifAddPending = uint8(0x01)
	pifDelPending = uint8(0x02)
	pifDisabled   = uint8(0x08)
)

// peerInfo collects stores and calculates moving average of last [SYNTROPY_PEERCHECK_WINDOW] link measurement
type peerInfo struct {
	publicKey    string
	connectionID int
	ifname       string
	flags        uint8
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

func (node *peerInfo) Valid() bool {
	// Ignore pifPending - not yet set, and pifDisabled - IP conflicting struct
	if node.flags != pifNone {
		return false
	}

	var sumLatency float32
	var sumLoss float32
	for _, val := range node.latency {
		sumLatency = sumLatency + val
	}
	for _, val := range node.loss {
		sumLoss = sumLoss + val
	}
	return (sumLatency > 0) || (sumLoss > 0)
}

func (node *peerInfo) IsPublic() bool {
	return strings.HasSuffix(node.ifname, env.InterfaceNamePublicSuffix)
}

func (node *peerInfo) String1() string {
	return fmt.Sprintf("dev %s loss: %f%% latency %fms",
		node.ifname, 100*node.Loss(), node.Latency())
}

func (node *peerInfo) HasFlag(f uint8) bool {
	return node.flags&f == f
}
