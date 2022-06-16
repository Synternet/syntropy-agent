package swireguard

import (
	"time"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const mega = 10000000

type PeerStats struct {
	TxBytesTotal  int64
	RxBytesTotal  int64
	TxBytesDiff   int64
	RxBytesDiff   int64
	TxSpeedMbps   float32
	RxSpeedMbps   float32
	LastHandshake time.Time
	timestamp     time.Time
}

func (ps *PeerStats) update(wgp *wgtypes.Peer, init bool) {
	// Calculate statistics first
	// First time run there will be no timestamp, so skip stats calculation
	if !ps.timestamp.IsZero() {
		if init {
			// init=true is passed to set initial counters.
			// Since this peer's statistics has been already calculated
			// we don't want to reset countes.
			// Continue stats calculation from current values
			return
		}

		ps.TxBytesDiff = wgp.TransmitBytes - ps.TxBytesTotal
		ps.RxBytesDiff = wgp.ReceiveBytes - ps.RxBytesTotal
		// TODO: overwrap handling ^^^
		// 100% loaded 10G link it will take ~467years, 100G - ~46years, 1T ~4.6years

		timeDiff := float32(time.Since(ps.timestamp) / time.Second)
		if timeDiff > 0 {
			ps.TxSpeedMbps = float32(ps.TxBytesDiff) / timeDiff / mega
			ps.RxSpeedMbps = float32(ps.RxBytesDiff) / timeDiff / mega
		}
	}

	// Then update Tx/Rx bytes for the next time
	ps.TxBytesTotal = wgp.TransmitBytes
	ps.RxBytesTotal = wgp.ReceiveBytes

	// timestamp, handshake, etc
	ps.timestamp = time.Now()
	ps.LastHandshake = wgp.LastHandshakeTime
}

func (wg *Wireguard) peerStatsCalculate(init bool) {
	for _, agentDev := range wg.devices {
		osDev, err := wg.wgc.Device(agentDev.IfName)
		if err != nil {
			continue
		}

		for _, osPeer := range osDev.Peers {
			for _, agentPeer := range agentDev.peers {
				if agentPeer.PublicKey == osPeer.PublicKey.String() {
					agentPeer.Stats.update(&osPeer, init)
					// skip to next peer on external loop
					break
				}
			}
		}
	}
}

func (wg *Wireguard) PeerStatsInit() {
	wg.peerStatsCalculate(true)
}

func (wg *Wireguard) PeerStatsUpdate() {
	wg.peerStatsCalculate(false)
}
