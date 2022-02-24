package ifacemon

import (
	"context"
	"log"
	"net"
	"strings"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/pkg/pubip"
	"github.com/vishvananda/netlink"
)

const (
	pkgName = "InterfaceMonitor. "
	cmd     = "IFACEMON"
)

type interfaceMonitor struct {
	publicIP net.IP
	t        *time.Ticker
	check    bool
}

func New() common.Service {
	return &interfaceMonitor{
		publicIP: pubip.GetPublicIp(),
		t:        time.NewTicker(10 * time.Second),
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
					log.Println(pkgName, "public IP change", obj.publicIP, "-->", pubip)
					obj.publicIP = pubip
				}
				obj.check = false
			}
		}
	}()

	return nil
}
