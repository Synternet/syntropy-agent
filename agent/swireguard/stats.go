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
	for _, sdev := range wg.devices {
		wdev, err := wg.wgc.Device(sdev.IfName)
		if err != nil {
			continue
		}

		for _, peer1 := range wdev.Peers {
			for _, peer2 := range sdev.peers {
				if peer2.PublicKey == peer1.PublicKey.String() {
					peer2.Stats.LastHandshake = peer1.LastHandshakeTime
					prevTx := peer2.Stats.TxBytes
					prevRx := peer2.Stats.RxBytes
					peer2.Stats.TxBytes = peer2.Stats.TxBytes + peer1.TransmitBytes
					peer2.Stats.RxBytes = peer2.Stats.RxBytes + peer1.ReceiveBytes
					if !peer2.Stats.timestamp.IsZero() {
						diff := float32(time.Since(peer2.Stats.timestamp) / time.Second)
						if diff > 0 {
							peer2.Stats.TxSpeedMbps = float32(peer2.Stats.TxBytes-prevTx) / diff / 1000000
							peer2.Stats.RxSpeedMbps = float32(peer2.Stats.RxBytes-prevRx) / diff / 1000000
						}
					}
					peer2.Stats.timestamp = time.Now()
					// skip to next peer on external loop
					break
				}
			}

		}
	}
}
