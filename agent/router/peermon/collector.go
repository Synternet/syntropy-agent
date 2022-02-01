package peermon

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	labels      = []string{"interface", "public_key", "internal_ip", "connection_id", "connection_group_id"}
	descLatency = prometheus.NewDesc(
		"latency",
		"Packet latency to connected peer",
		labels, nil,
	)
	descLoss = prometheus.NewDesc(
		"packet_loss",
		"Packet loss to connected peer",
		labels, nil,
	)
)

func (pm *PeerMonitor) Collect(ch chan<- prometheus.Metric, groupID int) {
	pm.RLock()
	defer pm.RUnlock()

	for _, peer := range pm.peerList {
		ch <- prometheus.MustNewConstMetric(
			descLatency,
			prometheus.GaugeValue,
			float64(peer.Latency()),
			peer.ifname, peer.publicKey, peer.endpoint, strconv.Itoa(peer.connectionID), strconv.Itoa(groupID),
		)
		ch <- prometheus.MustNewConstMetric(
			descLoss,
			prometheus.GaugeValue,
			float64(peer.Loss()),
			peer.ifname, peer.publicKey, peer.endpoint, strconv.Itoa(peer.connectionID), strconv.Itoa(groupID),
		)
	}
}
