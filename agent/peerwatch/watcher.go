package peerwatch

import (
	"context"
	"io"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/mole"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/multiping"
)

const (
	cmd     = "PEER_WATCHER"
	pkgName = "PeerWatcher. "
)

type wgPeerWatcher struct {
	writer             io.Writer
	mole               *mole.Mole
	pinger             *multiping.MultiPing
	pingData           *multiping.PingData
	counter            uint
	controlerSendCount uint
}

func New(writer io.Writer, m *mole.Mole, p *multiping.MultiPing) common.Service {
	return &wgPeerWatcher{
		mole:               m,
		writer:             writer,
		pinger:             p,
		pingData:           multiping.NewPingData(),
		controlerSendCount: uint(time.Minute / config.PeerCheckTime()),
	}
}

func (obj *wgPeerWatcher) execute(ctx context.Context) error {
	wgdevs := obj.mole.Wireguard().Devices()

	err := obj.monitorPeers(wgdevs)
	if err != nil {
		logger.Error().Println(pkgName, "monitor peers", err)
		return err
	}

	err = obj.message2controller(wgdevs)
	if err != nil {
		logger.Error().Println(pkgName, "send to controller", err)
		return err
	}

	return nil
}

func (obj *wgPeerWatcher) Name() string {
	return cmd
}

func (obj *wgPeerWatcher) Run(ctx context.Context) error {
	go func() {
		ticker := time.NewTicker(config.PeerCheckTime())
		// initial peer stats
		obj.mole.Wireguard().PeerStatsUpdate()
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Debug().Println(pkgName, "stopping", cmd)
				return
			case <-ticker.C:
				obj.execute(ctx)
			}
		}
	}()
	return nil
}
