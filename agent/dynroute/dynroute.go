// dynroute package actively monitores direct and sdn wireguard peers
// and setups best routing path
package dynroute

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/slock"
	"github.com/SyntropyNet/syntropy-agent-go/wireguard"
)

const (
	cmd         = "IFACES_PEERS_ACTIVE_DATA"
	pkgName     = "DynamicRouter. "
	checkPeriod = time.Second
)

type dynamicRouter struct {
	slock.AtomicServiceLock
	writer io.Writer
	wg     *wireguard.Wireguard
	ticker *time.Ticker
	stop   chan bool
}

func New(w io.Writer, wg *wireguard.Wireguard) common.Service {
	return &dynamicRouter{
		writer: w,
		wg:     wg,
		stop:   make(chan bool),
	}
}

func (obj *dynamicRouter) Name() string {
	return cmd
}

func (obj *dynamicRouter) execute() {
	log.Println(pkgName, "best route: ", obj.wg.Sdn().BestPath())
}

func (obj *dynamicRouter) Start() error {
	if !obj.TryLock() {
		return fmt.Errorf("dynamic routing already running")
	}

	obj.ticker = time.NewTicker(checkPeriod)
	go func() {
		for {
			select {
			case <-obj.stop:
				return
			case <-obj.ticker.C:
				obj.execute()

			}
		}
	}()

	return nil
}

func (obj *dynamicRouter) Stop() error {
	if !obj.TryUnlock() {
		return fmt.Errorf("dynamic routing is not running")
	}

	obj.ticker.Stop()
	obj.stop <- true

	return nil
}
