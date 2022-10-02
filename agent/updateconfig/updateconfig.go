package updateconfig

import (
	"encoding/json"
	"io"

	"github.com/SyntropyNet/syntropy-agent/agent/autoping"
	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/mole"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const (
	cmd     = "UPDATE_CONFIG"
	pkgName = "Config_Info. "
)

type configInfo struct {
	writer   io.Writer
	mole     *mole.Mole
	autoPing *autoping.AutoPing
}

func New(w io.Writer, m *mole.Mole, a *autoping.AutoPing) common.Command {
	return &configInfo{
		writer:   w,
		mole:     m,
		autoPing: a,
	}
}

func (obj *configInfo) Name() string {
	return cmd
}

func (obj *configInfo) Exec(raw []byte) error {
	var req common.ConfigMsg
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	for _, cmd := range req.Data.Peers {
		pi, err := cmd.AsPeerInfo()
		if err != nil {
			logger.Warning().Println(pkgName, err)
			continue
		}
		netpath, err := cmd.AsNetworkPath()
		if err != nil {
			logger.Warning().Println(pkgName, err)
			continue
		}
		switch cmd.Action {
		case "SET":
			err = obj.mole.AddPeer(pi, netpath)
			if err != nil {
				logger.Error().Println(pkgName, "Peers ", cmd.Action, err)
			}
		case "DEL":
			err = obj.mole.RemovePeer(pi, netpath)
		}
		if err != nil {
			logger.Error().Println(pkgName, "Peers ", cmd.Action, err)
		}
	}

	for _, cmd := range req.Data.Services {
		pi, err := cmd.AsServiceInfo()
		if err != nil {
			logger.Warning().Println(pkgName, err)
			continue
		}
		switch cmd.Action {
		case "SET":
			err = obj.mole.AddService(pi)
			if err != nil {
				logger.Error().Println(pkgName, "Services ", cmd.Action, err)
			}
		case "DEL":
			err = obj.mole.RemoveService(pi)
		}
	}

	for _, cmd := range req.Data.Settings.Rerouting {
		config.SetRerouteThresholds(cmd.ReroutingThreshold, cmd.LatencyCoefficient)
	}

	if len(req.Data.Settings.Autoping.IPs) > 0 {
		obj.autoPing.Exec(req.Data.Settings.Autoping)
	}

	// sync and merge everything between controller and OS
	obj.mole.Apply()

	return nil
}
