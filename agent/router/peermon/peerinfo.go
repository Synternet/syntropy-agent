package peermon

import "fmt"

// peerInfo collects stores and calculates moving average of last [valuesCount] link measurement
type peerInfo struct {
	endpoint string
	gateway  string
	latency  [valuesCount]float32
	loss     [valuesCount]float32
	index    int
}

func (node *peerInfo) Add(latency, loss float32) {
	node.latency[node.index] = latency
	node.loss[node.index] = loss
	node.index++
	if node.index >= valuesCount {
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
	for idx, val := range node.loss {
		if node.latency[idx] > 0 {
			sum = sum + val
			count++
		}
	}
	if count > 0 {
		return sum / float32(count)
	}
	return 0
}

func (node *peerInfo) String() string {
	return fmt.Sprintf("%s via %s loss: %f latency %f", node.endpoint, node.gateway, node.loss, node.latency)
}
