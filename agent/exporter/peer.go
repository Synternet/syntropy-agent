package exporter

import (
	"strconv"
	"sync"

	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
	"github.com/prometheus/client_golang/prometheus"
)

type peerInfo struct {
	Ifname       string
	PublicKey    string
	ConnectionID int
	GroupID      int
	Loss         float32
	Latency      float32
}

type peersCollector struct {
	sync.Mutex
	entries map[string]*peerInfo
}

func newPeersCollector() *peersCollector {
	return &peersCollector{
		entries: make(map[string]*peerInfo),
	}
}

func (pc *peersCollector) AddPeer(ip, ifname, pubkey string, connID, grID int) {
	pc.Lock()
	defer pc.Unlock()

	entry, ok := pc.entries[ip]
	if !ok {
		pc.entries[ip] = &peerInfo{
			Ifname:       ifname,
			PublicKey:    pubkey,
			ConnectionID: connID,
			GroupID:      grID,
		}
	} else if entry.PublicKey != pubkey || entry.Ifname != ifname {
		// Most probably peer was deleted and IP reused for other peer.
		// So parameters need to be updated
		pc.entries[ip] = &peerInfo{
			Ifname:       ifname,
			PublicKey:    pubkey,
			ConnectionID: connID,
			GroupID:      grID,
		}
	}
}

func (pc *peersCollector) PingProcess(pr *multiping.PingData) {
	pc.Lock()
	defer pc.Unlock()

	// Collect not updated IP addresses for removal later
	var removed []string

	// Process peers ping results
	for ip, peer := range pc.entries {
		pingRes, ok := pr.Get(ip)
		if !ok {
			removed = append(removed, ip)
		} else {
			peer.Latency = pingRes.Latency()
			peer.Loss = pingRes.Loss()
		}
	}

	// Remove outdated peers
	for _, ip := range removed {
		delete(pc.entries, ip)
	}
}

func (pc *peersCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(pc, ch)
}

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

func (pc *peersCollector) Collect(ch chan<- prometheus.Metric) {
	pc.Lock()
	defer pc.Unlock()

	for ip, peer := range pc.entries {
		ch <- prometheus.MustNewConstMetric(
			descLatency,
			prometheus.GaugeValue,
			float64(peer.Latency),
			peer.Ifname, peer.PublicKey, ip, strconv.Itoa(peer.ConnectionID), strconv.Itoa(peer.GroupID),
		)
		ch <- prometheus.MustNewConstMetric(
			descLoss,
			prometheus.GaugeValue,
			float64(peer.Loss),
			peer.Ifname, peer.PublicKey, ip, strconv.Itoa(peer.ConnectionID), strconv.Itoa(peer.GroupID),
		)
	}
}