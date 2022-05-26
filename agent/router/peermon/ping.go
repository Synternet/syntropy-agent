package peermon

import "github.com/SyntropyNet/syntropy-agent/pkg/multiping"

func (pm *PeerMonitor) PingProcess(pr *multiping.PingData) {
	pm.Lock()
	defer pm.Unlock()

	for _, peer := range pm.peerList {

		val, ok := pr.Get(peer.ip)
		if !ok {
			// NOTE: PeerMonitor is not creating its own ping list
			// It depends on other pingers and is an additional PingClient in their PingProces line
			// At first it may sound a bit complicate, but in fact it is not.
			// It just looks for its peers in other ping results. And it always founds its peers.
			// NOTE: Do not print error here - PeerMonitor always finds its peers. Just not all of them in one run.
			continue
		}
		peer.Add(val.Latency(), val.Loss())
	}

}
