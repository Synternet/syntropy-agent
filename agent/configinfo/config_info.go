package configinfo

import (
	"encoding/json"
	"io"

	"github.com/SyntropyNet/syntropy-agent/agent/docker"
	"github.com/SyntropyNet/syntropy-agent/agent/mole"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const (
	cmd     = "CONFIG_INFO"
	cmdResp = "UPDATE_AGENT_CONFIG"
	pkgName = "Config_Info. "
)

type configInfo struct {
	writer io.Writer
	mole   *mole.Mole
	docker docker.DockerHelper
}

func New(w io.Writer, m *mole.Mole, d docker.DockerHelper) common.Command {
	return &configInfo{
		writer: w,
		mole:   m,
		docker: d,
	}
}

func (obj *configInfo) Name() string {
	return cmd
}

func (obj *configInfo) Exec(raw []byte) error {
	var req configInfoMsg
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	resp := updateAgentConfigMsg{
		MessageHeader: req.MessageHeader,
		Data:          []updateAgentConfigEntry{},
	}
	resp.MsgType = cmdResp

	// CONFIG_INFO message sends me full configuration
	// Drop old cache and will build a new cache from zero
	obj.mole.Flush()

	// create missing interfaces
	wgi := req.Data.Network.Public.asInterfaceInfo("PUBLIC")
	err = obj.mole.CreateInterface(wgi)
	if err != nil {
		logger.Error().Printf("%s Create interface %s error: %s\n", pkgName, wgi.IfName, err)
	}
	if req.Data.Network.Public.PublicKey != wgi.PublicKey ||
		req.Data.Network.Public.Port != wgi.Port {
		resp.AddInterface(wgi)
	}

	wgi = req.Data.Network.Sdn1.asInterfaceInfo("SDN1")
	err = obj.mole.CreateInterface(wgi)
	if err != nil {
		logger.Error().Printf("%s Create interface %s error: %s\n", pkgName, wgi.IfName, err)
	}
	if req.Data.Network.Sdn1.PublicKey != wgi.PublicKey ||
		req.Data.Network.Sdn1.Port != wgi.Port {
		resp.AddInterface(wgi)
	}

	wgi = req.Data.Network.Sdn2.asInterfaceInfo("SDN2")
	err = obj.mole.CreateInterface(wgi)
	if err != nil {
		logger.Error().Printf("%s Create interface %s error: %s\n", pkgName, wgi.IfName, err)
	}
	if req.Data.Network.Sdn2.PublicKey != wgi.PublicKey ||
		req.Data.Network.Sdn2.Port != wgi.Port {
		resp.AddInterface(wgi)
	}

	wgi = req.Data.Network.Sdn3.asInterfaceInfo("SDN3")
	err = obj.mole.CreateInterface(wgi)
	if err != nil {
		logger.Error().Printf("%s Create interface %s error: %s\n", pkgName, wgi.IfName, err)
	}
	if req.Data.Network.Sdn3.PublicKey != wgi.PublicKey ||
		req.Data.Network.Sdn3.Port != wgi.Port {
		resp.AddInterface(wgi)
	}

	for _, subnetwork := range req.Data.Subnetworks {
		if subnetwork.Type == "DOCKER" {
			err := obj.docker.NetworkCreate(subnetwork.Name, subnetwork.Subnet)
			if err != nil {
				logger.Info().Printf("%s Docker subnetwork %s already created\n", pkgName, subnetwork.Name)
			}
		}
	}

	for _, cmd := range req.Data.VPN {
		switch cmd.Function {
		case "add_peer":
			err = obj.mole.AddPeer(cmd.asPeerInfo(), &common.SdnNetworkPath{
				Ifname:       cmd.Args.IfName,
				Gateway:      cmd.Args.GatewayIPv4,
				ConnectionID: cmd.Metadata.ConnectionID,
				GroupID:      cmd.Metadata.GroupID,
			})

		case "create_interface":
			wgi = cmd.asInterfaceInfo()
			err = obj.mole.CreateInterface(wgi)
			if err == nil &&
				cmd.Args.PublicKey != wgi.PublicKey ||
				cmd.Args.ListenPort != wgi.Port {
				resp.AddInterface(wgi)
			}
		}
		if err != nil {
			logger.Error().Println(pkgName, cmd.Function, err)
		}
	}

	resp.Now()
	arr, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	logger.Debug().Println(pkgName, "Sending: ", string(arr))
	obj.writer.Write(arr)

	// CONFIG_INFO message sends me full configuration
	// Finally sync and merge everything between controller and OS
	// (mostly for cleanup residual obsolete configuration)
	obj.mole.Apply()

	return nil
}
