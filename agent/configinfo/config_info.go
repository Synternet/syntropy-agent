package configinfo

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/config"
	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/SyntropyNet/syntropy-agent-go/logger"
	"github.com/SyntropyNet/syntropy-agent-go/wireguard"
)

const (
	cmd         = "CONFIG_INFO"
	cmdResp     = "UPDATE_AGENT_CONFIG"
	pkgName     = "Config_Info. "
	ifacePrefix = "SYNTROPY_"
)

type configInfo struct {
	writer io.Writer
	wg     *wireguard.Wireguard
}

type configInfoNetworkEntry struct {
	IP        string `json:"internal_ip"`
	PublicKey string `json:"public_key,omitempty"`
	Port      int    `json:"listen_port"`
}

func New(w io.Writer, wg *wireguard.Wireguard) controller.Command {
	return &configInfo{
		writer: w,
		wg:     wg,
	}
}

func (obj *configInfo) Name() string {
	return cmd
}

func (e *configInfoNetworkEntry) asInterfaceInfo(ifname string) *wireguard.InterfaceInfo {
	var name string
	if strings.HasPrefix(ifname, ifacePrefix) {
		name = ifname
	} else {
		name = ifacePrefix + ifname
	}
	return &wireguard.InterfaceInfo{
		IfName:    name,
		IP:        e.IP,
		PublicKey: e.PublicKey,
		Port:      e.Port,
	}
}

func (e *configInfoVpnEntry) asPeerInfo() *wireguard.PeerInfo {
	var name string
	if strings.HasPrefix(e.Args.IfName, ifacePrefix) {
		name = e.Args.IfName
	} else {
		name = ifacePrefix + e.Args.IfName
	}
	return &wireguard.PeerInfo{
		IfName:     name,
		IP:         e.Args.EndpointIPv4,
		PublicKey:  e.Args.PublicKey,
		Port:       e.Args.EndpointPort,
		Gateway:    e.Args.GatewayIPv4,
		AllowedIPs: e.Args.AllowedIPs,
	}
}

func (e *configInfoVpnEntry) asInterfaceInfo() *wireguard.InterfaceInfo {
	var name string
	if strings.HasPrefix(e.Args.IfName, ifacePrefix) {
		name = e.Args.IfName
	} else {
		name = ifacePrefix + e.Args.IfName
	}
	return &wireguard.InterfaceInfo{
		IfName:    name,
		IP:        e.Args.InternalIP,
		PublicKey: e.Args.PublicKey,
		Port:      e.Args.ListenPort,
	}
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
	} `json:"metadata,omitempty"`
}

type configInfoMsg struct {
	controller.MessageHeader
	Data struct {
		AgentID int `json:"agent_id"`
		Network struct {
			Public configInfoNetworkEntry `json:"PUBLIC"`
			Sdn1   configInfoNetworkEntry `json:"SDN1"`
			Sdn2   configInfoNetworkEntry `json:"SDN2"`
			Sdn3   configInfoNetworkEntry `json:"SDN3"`
		}
		VPN []configInfoVpnEntry `json:"vpn,omitempty"`
	} `json:"data"`
}

type updateAgentConfigEntry struct {
	Function string `json:"fn"`
	Data     struct {
		IfName    string `json:"ifname"`
		PublicKey string `json:"public_key"`
		IP        string `json:"internal_ip,omitempty"`
		Port      int    `json:"listen_port,omitempty"`
	} `json:"data"`
}

type updateAgentConfigMsg struct {
	controller.MessageHeader
	Data []updateAgentConfigEntry `json:"data"`
}

func (msg *updateAgentConfigMsg) AddInterface(data *wireguard.InterfaceInfo) {
	e := updateAgentConfigEntry{Function: "create_interface"}
	e.Data.IfName = data.IfName
	e.Data.IP = data.IP
	e.Data.PublicKey = data.PublicKey
	e.Data.Port = data.Port

	msg.Data = append(msg.Data, e)
}

func (msg *updateAgentConfigMsg) AddPeer(data *wireguard.PeerInfo) {
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

	// Dump pretty idented json to temp file
	// TODO: Do I need this file ??
	prettyJson, err := json.MarshalIndent(req, "", "    ")
	if err != nil {
		logger.Error().Println(pkgName, "json.MarshalIdent: ", err)
		return err
	}
	os.WriteFile(config.AgentTempDir+"/config_dump", prettyJson, 0600)

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

	for _, cmd := range req.Data.VPN {
		switch cmd.Function {
		case "add_peer":
			err = obj.wg.AddPeer(cmd.asPeerInfo())
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

	if errorCount > 0 {
		errResp := controller.ErrorResponce{
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
		obj.writer.Write(arr)
		return nil
	}

	resp.Now()
	arr, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	obj.writer.Write(arr)

	return nil
}
