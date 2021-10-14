// router package is used to setup routes
// also actively monitores direct and (sdn) wireguard peers
// and setups best routing path
package router

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/peermon"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/scontext"
)

const (
	cmd         = "IFACES_PEERS_ACTIVE_DATA"
	checkPeriod = time.Second * 3
	pkgName     = "Router. "
)

type peerActiveDataEntry struct {
	ConnectionID int    `json:"connection_id"`
	GroupID      int    `json:"connection_group_id"`
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
	writer      io.Writer
	peerMonitor *peermon.PeerMonitor
	ctx         scontext.StartStopContext

	routes map[string]*routeList
}

func New(ctx context.Context, w io.Writer, pm *peermon.PeerMonitor) *Router {
	return &Router{
		writer:      w,
		peerMonitor: pm,
		routes:      make(map[string]*routeList),
		ctx:         scontext.New(ctx),
	}
}

func (obj *Router) Name() string {
	return cmd
}

func (obj *Router) execute() {
	obj.Reroute(obj.peerMonitor.BestPath())
}

func (obj *Router) Start() error {
	ctx, err := obj.ctx.CreateContext()
	if err != nil {
		return fmt.Errorf("dynamic routing already running")
	}

	go func() {
		ticker := time.NewTicker(checkPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				obj.execute()

			}
		}
	}()

	return nil
}

func (obj *Router) Stop() error {
	if err := obj.ctx.CancelContext(); err != nil {
		return fmt.Errorf("dynamic routing is not running")
	}

	return nil
}
