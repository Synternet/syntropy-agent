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

	addPeerCount := 0
	delPeerCount := 0
	for _, cmd := range req.Data {
		switch cmd.Function {
		case "add_peer":
			pi, err := cmd.asPeerInfo()
			if err != nil {
				logger.Warning().Println(pkgName, err)
				continue
			}
			netpath, err := cmd.asNetworkPath()
			if err != nil {
				logger.Warning().Println(pkgName, err)
				continue
			}
			err = obj.mole.AddPeer(pi, netpath)
			if err == nil {
				addPeerCount++
			}

		case "remove_peer":
			pi, err := cmd.asPeerInfo()
			if err != nil {
				logger.Warning().Println(pkgName, err)
				continue
			}
			netpath, err := cmd.asNetworkPath()
			if err != nil {
				logger.Warning().Println(pkgName, err)
				continue
			}
			err = obj.mole.RemovePeer(pi, netpath)
			if err == nil {
				delPeerCount++
			}

		}

		if err != nil {
			logger.Error().Println(pkgName, cmd.Function, err)
		}
	}

	logger.Info().Println(pkgName, "Added:", addPeerCount, " Deleted:", delPeerCount, "peers")
	// sync and merge everything between controller and OS
	obj.mole.Apply()

	return nil
}
