// dynroute package actively monitores direct and sdn wireguard peers
// and setups best routing path
package dynroute

import (
	"io"

	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/SyntropyNet/syntropy-agent-go/wireguard"
)

const (
	cmd     = "IFACES_PEERS_ACTIVE_DATA"
	pkgName = "DynamicRouter. "
)

type dynamicRouter struct {
	writer io.Writer
	wg     wireguard.Wireguard
}

func New(w io.Writer, wg wireguard.Wireguard) controller.Service {
	return &dynamicRouter{
		writer: w,
		wg:     wg,
	}
}

func (obj *dynamicRouter) Name() string {
	return cmd
}

func (obj *dynamicRouter) Start() error {
	return nil
}

func (obj *dynamicRouter) Stop() error {
	return nil
}
