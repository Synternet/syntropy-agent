package configinfo

import (
	"encoding/json"
	"io"
	"os"

	"github.com/SyntropyNet/syntropy-agent/agent/docker"
	"github.com/SyntropyNet/syntropy-agent/agent/mole"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
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

func (obj *configInfo) processInterface(e *configInfoNetworkEntry, name string, resp *updateAgentConfigMsg) {
	if e == nil {
		return
	}
	wgi, err := e.asInterfaceInfo(name)
	if err != nil {
		logger.Error().Println(pkgName, "parse network", name, "failed", err)
		return
	}
	err = obj.mole.CreateInterface(wgi)
	if err != nil {
		logger.Error().Printf("%s Create interface %s error: %s\n", pkgName, wgi.IfName, err)
	}

	if e.PublicKey != wgi.PublicKey || e.Port != wgi.Port {
		resp.AddInterface(wgi)
	}
}

func (obj *configInfo) Exec(raw []byte) error {
	var req configInfoMsg
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	// Network section is empty is a special case
	// agent is deleted in the UI
	if req.Data.Network.Public == nil &&
		req.Data.Network.Sdn1 == nil &&
		req.Data.Network.Sdn2 == nil &&
		req.Data.Network.Sdn3 == nil {
		logger.Info().Println(pkgName, "Platform Agent deletion in progress.")

		// Cleanup will be done on mole, wireguard and router Close functions.
		// But CleanupOnExit must be enabled. Force enable it
		config.ForceCleanupOnExit()

		// First try sending SIGTERM signal to self and it will cleanup all on exit
		process, err := os.FindProcess(os.Getpid())
		if err == nil {
			logger.Debug().Println(pkgName, "Sending SIGINT signal to self process")
			process.Signal(os.Interrupt)
			// exit and signal handler will complete termination
			return nil
		}

		// Self process finding failed. Can it ever happen ?
		logger.Error().Println(pkgName, "Platform Agent exit failed getting self pid", err)
		logger.Error().Println(pkgName, "Exiting anyway")
		// mole.Close should do most cleanup
		obj.mole.Close()
		os.Exit(0)
	}

	resp := &updateAgentConfigMsg{
		MessageHeader: req.MessageHeader,
		Data:          []updateAgentConfigEntry{},
	}
	resp.MsgType = cmdResp

	// CONFIG_INFO can be quite big message and could take a longer time to process
	// Thus note that processing has started
	logger.Info().Println(pkgName, "Configuring...")
	// CONFIG_INFO message sends me full configuration
	// Drop old cache and will build a new cache from zero
	obj.mole.Flush()

	// create missing interfaces
	obj.processInterface(req.Data.Network.Public, "PUBLIC", resp)
	obj.processInterface(req.Data.Network.Sdn1, "SDN1", resp)
	obj.processInterface(req.Data.Network.Sdn2, "SDN2", resp)
	obj.processInterface(req.Data.Network.Sdn3, "SDN3", resp)

	for _, subnetwork := range req.Data.Subnetworks {
		if subnetwork.Type == "DOCKER" {
			err := obj.docker.NetworkCreate(subnetwork.Name, subnetwork.Subnet)
			if err != nil {
				logger.Info().Printf("%s Docker subnetwork %s already created\n", pkgName, subnetwork.Name)
			}
		}
	}

	addPeerCount := 0
	for _, cmd := range req.Data.VPN {
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

		case "create_interface":
			wgi, err := cmd.asInterfaceInfo()
			if err != nil {
				logger.Error().Println(pkgName, "parse interface info failed", err)
				continue
			}
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

	logger.Info().Println(pkgName, "Configured", addPeerCount, "peers")
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
