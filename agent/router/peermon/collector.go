package peermon

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	labels      = []string{"interface", "public_key", "internal_ip", "connection_id", "connection_group_id"}
	descLatency = prometheus.NewDesc(
		"syntropy_platform_latency",
		"Packet latency to connected peer",
		labels, nil,
	)
	descLoss = prometheus.NewDesc(
		"syntropy_platform_packet_loss",
		"Packet loss to connected peer",
		labels, nil,
	)
)

func (pm *PeerMonitor) Collect(ch chan<- prometheus.Metric, groupID int) {
	for addr, peer := range pm.peerList {
		ch <- prometheus.MustNewConstMetric(
			descLatency,
			prometheus.GaugeValue,
			float64(peer.Latency()),
			peer.ifname, peer.publicKey, addr.Addr().String(), strconv.Itoa(peer.connectionID), strconv.Itoa(groupID),
		)
		ch <- prometheus.MustNewConstMetric(
			descLoss,
			prometheus.GaugeValue,
			float64(peer.Loss()),
			peer.ifname, peer.publicKey, addr.Addr().String(), strconv.Itoa(peer.connectionID), strconv.Itoa(groupID),
		)
	}
}
