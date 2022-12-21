package peermon

import (
	"net/netip"
	"strconv"

	"github.com/SyntropyNet/syntropy-agent/agent/router/peermon/peerlist"
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
	pm.peerList.Iterate(func(addr netip.Prefix, peer *peerlist.PeerInfo) {
		ch <- prometheus.MustNewConstMetric(
			descLatency,
			prometheus.GaugeValue,
			float64(peer.Latency()),
			peer.Ifname, peer.PublicKey, addr.Addr().String(), strconv.Itoa(peer.ConnectionID), strconv.Itoa(groupID),
		)
		ch <- prometheus.MustNewConstMetric(
			descLoss,
			prometheus.GaugeValue,
			float64(peer.Loss()),
			peer.Ifname, peer.PublicKey, addr.Addr().String(), strconv.Itoa(peer.ConnectionID), strconv.Itoa(groupID),
		)
	})
}
