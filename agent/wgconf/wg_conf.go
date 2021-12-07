package wgconf

import (
	"encoding/json"
	"io"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/mole"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const (
	pkgName = "Wg_Conf. "
	cmd     = "WG_CONF"
)

type wgConf struct {
	writer io.Writer
	mole   *mole.Mole
}

func New(w io.Writer, m *mole.Mole) common.Command {
	return &wgConf{
		writer: w,
		mole:   m,
	}
}

func (obj *wgConf) Name() string {
	return cmd
}

func (obj *wgConf) Exec(raw []byte) error {
	var req wgConfMsg
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	for _, cmd := range req.Data {
		switch cmd.Function {
		case "add_peer":
			wgp := cmd.asPeerInfo()
			err = obj.mole.AddPeer(wgp, &common.SdnNetworkPath{
				Ifname:       cmd.Args.IfName,
				Gateway:      cmd.Args.GatewayIPv4,
				ConnectionID: cmd.Metadata.ConnectionID,
				GroupID:      cmd.Metadata.GroupID,
			})

		case "remove_peer":
			wgp := cmd.asPeerInfo()
			err = obj.mole.RemovePeer(wgp, &common.SdnNetworkPath{
				Ifname: cmd.Args.IfName,
			})
		}

		if err != nil {
			logger.Error().Println(pkgName, cmd.Function, err)
		}
	}

	// sync and merge everything between controller and OS
	obj.mole.Apply()

	return nil
}
