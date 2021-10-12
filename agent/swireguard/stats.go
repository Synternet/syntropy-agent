package swireguard

import (
	"time"
)

type PeerStats struct {
	TxBytes       int64
	RxBytes       int64
	TxSpeedMbps   float32
	RxSpeedMbps   float32
	LastHandshake time.Time
	timestamp     time.Time
}

func (wg *Wireguard) PeerStatsUpdate() {
	for _, agentDev := range wg.devices {
		osDev, err := wg.wgc.Device(agentDev.IfName)
		if err != nil {
			continue
		}

		for _, osPeer := range osDev.Peers {
			for _, agentPeer := range agentDev.peers {
				if agentPeer.PublicKey == osPeer.PublicKey.String() {
					// Calculate statistics first
					if !agentPeer.Stats.timestamp.IsZero() {
						diff := float32(time.Since(agentPeer.Stats.timestamp) / time.Second)
						if diff > 0 {
							agentPeer.Stats.TxSpeedMbps = float32(osPeer.TransmitBytes-agentPeer.Stats.TxBytes) / diff / 1000000
							agentPeer.Stats.RxSpeedMbps = float32(osPeer.ReceiveBytes-agentPeer.Stats.RxBytes) / diff / 1000000
						}
					}
					// Then update Tx/Rx bytes for the next time
					agentPeer.Stats.TxBytes = osPeer.TransmitBytes
					agentPeer.Stats.RxBytes = osPeer.ReceiveBytes

					// timestamp, handshake, etc
					agentPeer.Stats.timestamp = time.Now()
					agentPeer.Stats.LastHandshake = osPeer.LastHandshakeTime
					// skip to next peer on external loop
					break
				}
			}

		}
	}
}
