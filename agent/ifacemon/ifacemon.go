package ifacemon

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/pubip"
	"github.com/vishvananda/netlink"
)

const (
	pkgName = "InterfaceMonitor. "
	cmd     = "IFACEMON"
)

type interfaceMonitor struct {
	publicIP  net.IP
	t         *time.Ticker
	check     bool
	reconnect func() error
}

func New(f func() error) common.Service {
	return &interfaceMonitor{
		publicIP:  pubip.GetPublicIp(),
		t:         time.NewTicker(10 * time.Second),
		reconnect: f,
	}
}

func (obj *interfaceMonitor) Name() string {
	return cmd
}

func (obj *interfaceMonitor) Run(ctx context.Context) error {
	var update netlink.LinkUpdate
	ch := make(chan netlink.LinkUpdate)
	done := make(chan struct{})

	// Listen to interface state change events
	err := netlink.LinkSubscribe(ch, done)
	if err != nil {
		close(done)
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Debug().Println(pkgName, "stopping", cmd)
				close(done)
				return

			case update = <-ch:
				// Ignore self created interfaces change
				if strings.Contains(update.Attrs().Name, env.InterfaceNamePrefix) {
					continue
				}
				// check event in separate timer
				obj.check = true

			case <-obj.t.C:
				if !obj.check {
					continue
				}

				// forced getting public IP address
				pubip.Reset()
				pubip := pubip.GetPublicIp()
				if pubip == nil || pubip.IsUnspecified() {
					// no public IP yet. Retry later
					continue
				}

				// check if public IP has changed
				if !pubip.Equal(obj.publicIP) {
					logger.Info().Println(pkgName, "public IP change", obj.publicIP, "-->", pubip)
					obj.publicIP = pubip
					if obj.reconnect != nil {
						logger.Info().Println(pkgName, "Forcing reconnection")
						err := obj.reconnect()
						if err != nil {
							logger.Error().Println(pkgName, "Reconnect on public IP change", err)
						}
					}
				}
				obj.check = false
			}
		}
	}()

	return nil
}
