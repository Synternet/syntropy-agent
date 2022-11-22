package setconfig

import (
	"encoding/json"
	"github.com/SyntropyNet/syntropy-agent/agent/autoping"
	"io"
	"os"

	"github.com/SyntropyNet/syntropy-agent/agent/docker"
	"github.com/SyntropyNet/syntropy-agent/agent/mole"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const (
	cmd     = "SET_CONFIG"
	cmdResp = "CONFIG_INFO"
	pkgName = "Config_Info. "
)

type configInfo struct {
	writer   io.Writer
	mole     *mole.Mole
	autoPing *autoping.AutoPing
	docker   docker.DockerHelper
}

func New(w io.Writer, m *mole.Mole, a *autoping.AutoPing, d docker.DockerHelper) common.Command {
	return &configInfo{
		writer:   w,
		mole:     m,
		autoPing: a,
		docker:   d,
	}
}

func (obj *configInfo) Name() string {
	return cmd
}

func (obj *configInfo) processInterface(e *common.ConfigInterfaceEntry, resp *ConfigInfoMsg) {
	if e == nil {
		return
	}
	wgi, err := e.AsInterfaceInfo()
	if err != nil {
		logger.Error().Println(pkgName, "parse network", " INDEX:", e.Index, "failed", err)
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
	var req common.ConfigMsg
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	// Network section is empty is a special case
	// agent is deleted in the UI
	if len(req.Data.Interfaces) == 0 {
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

	resp := &ConfigInfoMsg{
		MessageHeader: req.MessageHeader,
		Data: ConfigInfoEntry{
			Interfaces: []InterfaceEntry{},
		},
	}
	resp.MsgType = cmdResp

	// SET_CONFIG can be quite big message and could take a longer time to process
	// Thus note that processing has started
	logger.Info().Println(pkgName, "Configuring...")
	// SET_CONFIG message sends me full configuration
	// Drop old cache and will build a new cache from zero
	obj.mole.Flush()

	// create missing interfaces
	for _, iface := range req.Data.Interfaces {
		obj.processInterface(&iface, resp)
	}

	for _, subnetwork := range req.Data.Subnetworks {
		if subnetwork.Type == "DOCKER" {
			err := obj.docker.NetworkCreate(subnetwork.Name, subnetwork.Subnet)
			if err != nil {
				logger.Info().Printf("%s Docker subnetwork %s already created\n", pkgName, subnetwork.Name)
			}
		}
	}

	addPeerCount := 0
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
		err = obj.mole.AddPeer(pi, netpath)
		if err == nil {
			addPeerCount++
		} else {
			logger.Error().Println(pkgName, "Peers ", cmd.Action, err)
		}
	}

	for _, cmd := range req.Data.Services {
		pi, err := cmd.AsServiceInfo()
		if err != nil {
			logger.Warning().Println(pkgName, err)
			continue
		}
		err = obj.mole.AddService(pi)
		if err != nil {
			logger.Error().Println(pkgName, "Services ", cmd.Action, err)
		}
	}

	for _, cmd := range req.Data.Settings.Rerouting {
		config.SetRerouteThresholds(cmd.ReroutingThreshold, cmd.LatencyCoefficient)
	}
	if len(req.Data.Settings.Autoping.IPs) > 0 {
		obj.autoPing.Exec(req.Data.Settings.Autoping)
	}

	logger.Info().Println(pkgName, "Configured", addPeerCount, "peers")
	resp.Now()
	arr, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	logger.Message().Println(pkgName, "Sending: ", string(arr))
	obj.writer.Write(arr)

	// SET_CONFIG message sends me full configuration
	// Finally sync and merge everything between controller and OS
	// (mostly for cleanup residual obsolete configuration)
	obj.mole.Apply()

	return nil
}
