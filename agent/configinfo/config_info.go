package configinfo

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/agent/docker"
	"github.com/SyntropyNet/syntropy-agent-go/agent/peeradata"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/agent/router"
	"github.com/SyntropyNet/syntropy-agent-go/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent-go/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
)

const (
	cmd     = "CONFIG_INFO"
	cmdResp = "UPDATE_AGENT_CONFIG"
	pkgName = "Config_Info. "
)

type configInfo struct {
	writer io.Writer
	wg     *swireguard.Wireguard
	router *router.Router
	docker docker.DockerHelper
}

type configInfoNetworkEntry struct {
	IP        string `json:"internal_ip"`
	PublicKey string `json:"public_key,omitempty"`
	Port      int    `json:"listen_port,omitempty"`
}

func New(w io.Writer, wg *swireguard.Wireguard, r *router.Router, d docker.DockerHelper) common.Command {
	return &configInfo{
		writer: w,
		wg:     wg,
		router: r,
		docker: d,
	}
}

func (obj *configInfo) Name() string {
	return cmd
}

func (e *configInfoNetworkEntry) asInterfaceInfo(ifaceName string) *swireguard.InterfaceInfo {
	var ifname string
	if strings.HasPrefix(ifaceName, env.InterfaceNamePrefix) {
		ifname = ifaceName
	} else {
		ifname = env.InterfaceNamePrefix + ifaceName
	}
	return &swireguard.InterfaceInfo{
		IfName:    ifname,
		IP:        e.IP,
		PublicKey: e.PublicKey,
		Port:      e.Port,
	}
}

func (e *configInfoVpnEntry) asPeerInfo() *swireguard.PeerInfo {
	var ifname string
	if strings.HasPrefix(e.Args.IfName, env.InterfaceNamePrefix) {
		ifname = e.Args.IfName
	} else {
		ifname = env.InterfaceNamePrefix + e.Args.IfName
	}
	return &swireguard.PeerInfo{
		IfName:       ifname,
		IP:           e.Args.EndpointIPv4,
		PublicKey:    e.Args.PublicKey,
		ConnectionID: e.Metadata.ConnectionID,
		GroupID:      e.Metadata.GroupID,
		AgentID:      e.Metadata.AgentID,
		Port:         e.Args.EndpointPort,
		Gateway:      e.Args.GatewayIPv4,
		AllowedIPs:   e.Args.AllowedIPs,
	}
}

func (e *configInfoVpnEntry) asInterfaceInfo() *swireguard.InterfaceInfo {
	var ifname string
	if strings.HasPrefix(e.Args.IfName, env.InterfaceNamePrefix) {
		ifname = e.Args.IfName
	} else {
		ifname = env.InterfaceNamePrefix + e.Args.IfName
	}
	return &swireguard.InterfaceInfo{
		IfName:    ifname,
		IP:        e.Args.InternalIP,
		PublicKey: e.Args.PublicKey,
		Port:      e.Args.ListenPort,
	}
}

type configInfoSubnetworksEntry struct {
	Name   string `json:"name"`
	Subnet string `json:"subnet"`
	Type   string `json:"type"`
}

type configInfoVpnEntry struct {
	Function string `json:"fn"`

	Args struct {
		// Common fields
		IfName    string `json:"ifname"`
		PublicKey string `json:"public_key,omitempty"`
		// create_interface
		InternalIP string `json:"internal_ip,omitempty"`
		ListenPort int    `json:"listen_port,omitempty"`
		// add_peer
		AllowedIPs   []string `json:"allowed_ips,omitempty"`
		EndpointIPv4 string   `json:"endpoint_ipv4,omitempty"`
		EndpointPort int      `json:"endpoint_port,omitempty"`
		GatewayIPv4  string   `json:"gw_ipv4,omitempty"`
	} `json:"args,omitempty"`

	Metadata struct {
		// create_interface
		NetworkID int `json:"network_id,omitempty"`
		// add_peer
		DeviceID         string `json:"device_id,omitempty"`
		DeviceName       string `json:"device_name,omitempty"`
		DevicePublicIPv4 string `json:"device_public_ipv4,omitempty"`
		ConnectionID     int    `json:"connection_id,omitempty"`
		GroupID          int    `json:"connection_group_id,omitempty"`
		AgentID          int    `json:"agent_id,omitempty"`
	} `json:"metadata,omitempty"`
}

type configInfoMsg struct {
	common.MessageHeader
	Data struct {
		AgentID int `json:"agent_id"`
		Network struct {
			Public configInfoNetworkEntry `json:"PUBLIC"`
			Sdn1   configInfoNetworkEntry `json:"SDN1"`
			Sdn2   configInfoNetworkEntry `json:"SDN2"`
			Sdn3   configInfoNetworkEntry `json:"SDN3"`
		}
		VPN         []configInfoVpnEntry         `json:"vpn,omitempty"`
		Subnetworks []configInfoSubnetworksEntry `json:"subnetworks,omitempty"`
	} `json:"data"`
}

type updateAgentConfigEntry struct {
	Function string `json:"fn"`
	Data     struct {
		IfName    string `json:"ifname"`
		PublicKey string `json:"public_key"`
		IP        string `json:"internal_ip"`
		Port      int    `json:"listen_port"`
	} `json:"data"`
}

type updateAgentConfigMsg struct {
	common.MessageHeader
	Data []updateAgentConfigEntry `json:"data"`
}

func (msg *updateAgentConfigMsg) AddInterface(data *swireguard.InterfaceInfo) {
	e := updateAgentConfigEntry{Function: "create_interface"}
	e.Data.IfName = data.IfName
	e.Data.IP = data.IP
	e.Data.PublicKey = data.PublicKey
	e.Data.Port = data.Port

	msg.Data = append(msg.Data, e)
}

func (msg *updateAgentConfigMsg) AddPeer(data *swireguard.PeerInfo) {
	e := updateAgentConfigEntry{Function: "add_peer"}
	e.Data.IfName = data.IfName
	e.Data.IP = data.IP
	e.Data.PublicKey = data.PublicKey
	e.Data.Port = data.Port

	msg.Data = append(msg.Data, e)
}

func (obj *configInfo) Exec(raw []byte) error {
	var req configInfoMsg
	var errorCount int
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	resp := updateAgentConfigMsg{
		MessageHeader: req.MessageHeader,
		Data:          []updateAgentConfigEntry{},
	}
	resp.MsgType = cmdResp

	routeStatus := routestatus.NewMsg()
	padMsg := peeradata.NewMessage()

	// CONFIG_INFO message sends me full configuration
	// Drop old cache and will build a new cache from zero
	obj.wg.Flush()

	// create missing interfaces
	wgi := req.Data.Network.Public.asInterfaceInfo("PUBLIC")
	err = obj.wg.CreateInterface(wgi)
	if err != nil {
		logger.Error().Printf("%s Create interface %s error: %s\n", pkgName, wgi.IfName, err)
		errorCount++
	}
	if req.Data.Network.Public.PublicKey != wgi.PublicKey ||
		req.Data.Network.Public.Port != wgi.Port {
		resp.AddInterface(wgi)
	}

	wgi = req.Data.Network.Sdn1.asInterfaceInfo("SDN1")
	err = obj.wg.CreateInterface(wgi)
	if err != nil {
		logger.Error().Printf("%s Create interface %s error: %s\n", pkgName, wgi.IfName, err)
		errorCount++
	}
	if req.Data.Network.Sdn1.PublicKey != wgi.PublicKey ||
		req.Data.Network.Sdn1.Port != wgi.Port {
		resp.AddInterface(wgi)
	}

	wgi = req.Data.Network.Sdn2.asInterfaceInfo("SDN2")
	err = obj.wg.CreateInterface(wgi)
	if err != nil {
		logger.Error().Printf("%s Create interface %s error: %s\n", pkgName, wgi.IfName, err)
		errorCount++
	}
	if req.Data.Network.Sdn2.PublicKey != wgi.PublicKey ||
		req.Data.Network.Sdn2.Port != wgi.Port {
		resp.AddInterface(wgi)
	}

	wgi = req.Data.Network.Sdn3.asInterfaceInfo("SDN3")
	err = obj.wg.CreateInterface(wgi)
	if err != nil {
		logger.Error().Printf("%s Create interface %s error: %s\n", pkgName, wgi.IfName, err)
		errorCount++
	}
	if req.Data.Network.Sdn3.PublicKey != wgi.PublicKey ||
		req.Data.Network.Sdn3.Port != wgi.Port {
		resp.AddInterface(wgi)
	}

	for _, subnetwork := range req.Data.Subnetworks {
		if subnetwork.Type == "DOCKER" {
			err := obj.docker.NetworkCreate(subnetwork.Name, subnetwork.Subnet)
			if err != nil {
				logger.Error().Printf("%s Create docker network error: %s\n", pkgName, err)
			}
		}
	}

	for _, cmd := range req.Data.VPN {
		switch cmd.Function {
		case "add_peer":
			err = obj.wg.AddPeer(cmd.asPeerInfo())
			if err == nil {
				routeRes, peersData := obj.router.RouteAdd(
					&common.SdnNetworkPath{
						Ifname:       cmd.Args.IfName,
						Gateway:      cmd.Args.GatewayIPv4,
						ConnectionID: cmd.Metadata.ConnectionID,
						GroupID:      cmd.Metadata.GroupID,
					}, cmd.Args.AllowedIPs)
				routeStatus.Add(cmd.Metadata.ConnectionID, cmd.Metadata.GroupID, routeRes)
				padMsg.Add(peersData...)
			}

		case "create_interface":
			wgi = cmd.asInterfaceInfo()
			err = obj.wg.CreateInterface(wgi)
			if err == nil &&
				cmd.Args.PublicKey != wgi.PublicKey ||
				cmd.Args.ListenPort != wgi.Port {
				resp.AddInterface(wgi)
			}
		}
		if err != nil {
			logger.Error().Println(pkgName, cmd.Function, err)
			errorCount++
		}
	}

	// CONFIG_INFO message sends me full configuration
	// Now sync and merge everything between controller and OS
	// (mostly for cleanup residual obsolete configuration)
	err = obj.wg.Apply()
	if err != nil {
		logger.Error().Println(pkgName, "wireguard apply", err)
	}

	if errorCount > 0 {
		errResp := common.ErrorResponce{
			MessageHeader: req.MessageHeader,
		}
		errResp.Data.Type = cmd + "_ERROR"
		errResp.Data.Message = fmt.Sprintf("There were %d errors while performing %s request %s",
			errorCount, req.MsgType, req.ID)
		errResp.Now()
		arr, err := json.Marshal(errResp)
		if err != nil {
			return err
		}
		// Tricky here: I have errors, and I send them back to controller
		// But they are not internal application errors
		logger.Debug().Println(pkgName, "Sending: ", string(arr))
		obj.writer.Write(arr)
		return nil
	}

	resp.Now()
	arr, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	logger.Debug().Println(pkgName, "Sending: ", string(arr))
	obj.writer.Write(arr)

	routeStatus.Send(obj.writer)
	padMsg.Send(obj.writer)

	return nil
}
