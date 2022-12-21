package peerlist

import (
	"fmt"
	"strings"

	"github.com/SyntropyNet/syntropy-agent/internal/env"
)

const (
	PifNone       = uint8(0x00)
	PifAddPending = uint8(0x01)
	PifDelPending = uint8(0x02)
	PifDisabled   = uint8(0x08)
)

// PeerInfo collects stores and calculates moving average of last [SYNTROPY_PEERCHECK_WINDOW] link measurement
type PeerInfo struct {
	PublicKey    string
	ConnectionID int
	Ifname       string
	flags        uint8
	latency      []float32
	loss         []float32
	index        int
}

func NewPeerInfo(avgCount uint) *PeerInfo {
	pi := PeerInfo{
		latency: make([]float32, avgCount),
		loss:    make([]float32, avgCount),
	}
	return &pi
}

func (node *PeerInfo) Add(latency, loss float32) {
	node.latency[node.index] = latency
	node.loss[node.index] = loss
	node.index++
	if node.index >= cap(node.latency) {
		node.index = 0
	}
}

func (node *PeerInfo) Latency() float32 {
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

func (node *PeerInfo) Loss() float32 {
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

func (node *PeerInfo) StatsIncomplete() bool {
	count := 0
	for _, val := range node.latency {
		if val > 0 {
			count++
		}
	}
	return count != cap(node.latency)
}

func (node *PeerInfo) Valid() bool {
	// Ignore pifPending - not yet set, and pifDisabled - IP conflicting struct
	if node.flags != PifNone {
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

func (node *PeerInfo) IsPublic() bool {
	return strings.HasSuffix(node.Ifname, env.InterfaceNamePublicSuffix)
}

func (node *PeerInfo) String1() string {
	return fmt.Sprintf("dev %s loss: %f%% latency %fms",
		node.Ifname, 100*node.Loss(), node.Latency())
}

func (node *PeerInfo) SetFlag(f uint8) {
	node.flags |= f
}

func (node *PeerInfo) ClearFlag(f uint8) {
	node.flags &= ^f
}

func (node *PeerInfo) HasFlag(f uint8) bool {
	return node.flags&f == f
}

func (node *PeerInfo) ResetFlags() {
	node.flags = PifNone
}
