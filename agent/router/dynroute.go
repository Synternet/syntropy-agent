// router package is used to setup routes
// also actively monitores direct and (sdn) wireguard peers
// and setups best routing path
package router

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/peermon"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/slock"
)

const (
	cmd         = "IFACES_PEERS_ACTIVE_DATA"
	checkPeriod = time.Second * 3
	pkgName     = "Router. "
)

type peerActiveDataEntry struct {
	ConnectionID int    `json:"connection_id"`
	Timestamp    string `json:"timestamp"`
}

type peersActiveDataMessage struct {
	common.MessageHeader
	Data []peerActiveDataEntry `json:"data"`
}

func newRespMsg() *peersActiveDataMessage {
	resp := peersActiveDataMessage{
		Data: []peerActiveDataEntry{},
	}
	resp.ID = env.MessageDefaultID
	resp.MsgType = cmd
	return &resp
}

type Router struct {
	sync.Mutex
	slock.AtomicServiceLock
	writer      io.Writer
	peerMonitor *peermon.PeerMonitor
	ticker      *time.Ticker
	stop        chan bool

	routes map[string]*routeList
}

func New(w io.Writer, pm *peermon.PeerMonitor) *Router {
	return &Router{
		writer:      w,
		peerMonitor: pm,
		stop:        make(chan bool),
		routes:      make(map[string]*routeList),
	}
}

func (obj *Router) Name() string {
	return cmd
}

func (obj *Router) execute() {
	obj.Reroute(obj.peerMonitor.BestPath())
}

func (obj *Router) Start() error {
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

func (obj *Router) Stop() error {
	if !obj.TryUnlock() {
		return fmt.Errorf("dynamic routing is not running")
	}

	obj.ticker.Stop()
	obj.stop <- true

	return nil
}
