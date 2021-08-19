package agent

import (
	"encoding/json"
	"os"

	"github.com/SyntropyNet/syntropy-agent-go/config"
	"github.com/SyntropyNet/syntropy-agent-go/logger"
	"github.com/SyntropyNet/syntropy-agent-go/wireguard"
)

type configInfoNetworkEntry struct {
	IP        string `json:"internal_ip"`
	PublicKey string `json:"public_key,omitempty"`
	Port      int    `json:"listen_port"`
}

func (e *configInfoNetworkEntry) AsInterfaceInfo() *wireguard.InterfaceInfo {
	return &wireguard.InterfaceInfo{
		IP:        e.IP,
		PublicKey: e.PublicKey,
		Port:      e.Port,
	}
}

func (e *configInfoVpnEntry) AsPeerInfo() *wireguard.PeerInfo {
	return &wireguard.PeerInfo{
		IfName:     e.Args.IfName,
		IP:         e.Args.EndpointIPv4,
		PublicKey:  e.Args.PublicKey,
		Port:       e.Args.EndpointPort,
		Gateway:    e.Args.GatewayIPv4,
		AllowedIPs: e.Args.AllowedIPs,
	}
}

func (e *configInfoVpnEntry) AsInterfaceInfo() *wireguard.InterfaceInfo {
	return &wireguard.InterfaceInfo{
		IfName:    e.Args.IfName,
		IP:        e.Args.InternalIP,
		PublicKey: e.Args.PublicKey,
		Port:      e.Args.ListenPort,
	}
}

/****    TODO: review me      ******/
//	I'm not sure this is a good idea, but I wanted to decode json in one step
//	So I am mixing different structs in one instance
//	And will try to use only correct fields, depending on `fn` type
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
	messageHeader
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
	messageHeader
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

func configInfo(a *Agent, raw []byte) error {
	var req configInfoMsg
	var errorCount int
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	resp := updateAgentConfigMsg{
		messageHeader: req.messageHeader,
	}
	resp.MsgType = "UPDATE_AGENT_CONFIG"
	// Initialise empty slice, so if no entries is added
	// json.Marshal will result in empty json, and not a null object
	resp.Data = []updateAgentConfigEntry{}

	// Dump pretty idented json to temp file
	// TODO: Do I need this file ??
	prettyJson, err := json.MarshalIndent(req, "", "    ")
	if err != nil {
		return err
	}
	os.WriteFile(config.AgentTempDir+"/config_dump", prettyJson, 0600)

	// create missing interfaces
	wgi := req.Data.Network.Public.AsInterfaceInfo()
	wgi.IfName = "SYNTROPY_PUBLIC"
	err = a.wg.CreateInterface(wgi)
	if err != nil {
		return err
	}
	if req.Data.Network.Public.PublicKey != wgi.PublicKey ||
		req.Data.Network.Public.Port != wgi.Port {
		resp.AddInterface(wgi)
	}

	wgi = req.Data.Network.Sdn1.AsInterfaceInfo()
	wgi.IfName = "SYNTROPY_SDN1"
	err = a.wg.CreateInterface(wgi)
	if err != nil {
		return err
	}
	if req.Data.Network.Sdn1.PublicKey != wgi.PublicKey ||
		req.Data.Network.Sdn1.Port != wgi.Port {
		resp.AddInterface(wgi)
	}

	wgi = req.Data.Network.Sdn2.AsInterfaceInfo()
	wgi.IfName = "SYNTROPY_SDN2"
	err = a.wg.CreateInterface(wgi)
	if err != nil {
		return err
	}
	if req.Data.Network.Sdn2.PublicKey != wgi.PublicKey ||
		req.Data.Network.Sdn2.Port != wgi.Port {
		resp.AddInterface(wgi)
	}

	wgi = req.Data.Network.Sdn3.AsInterfaceInfo()
	wgi.IfName = "SYNTROPY_SDN3"
	err = a.wg.CreateInterface(wgi)
	if err != nil {
		return err
	}
	if req.Data.Network.Sdn3.PublicKey != wgi.PublicKey ||
		req.Data.Network.Sdn3.Port != wgi.Port {
		resp.AddInterface(wgi)
	}

	for _, cmd := range req.Data.VPN {
		switch cmd.Function {
		case "add_peer":
			err = a.wg.AddPeer(cmd.AsPeerInfo())
		case "create_interface":
			// TODO: need to rethink where and how to setup `routes` and `iptables` rules
			wgi = cmd.AsInterfaceInfo()
			err = a.wg.CreateInterface(wgi)
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
		// TODO: add error information to controller
	}

	resp.Now()
	arr, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	a.Write(arr)

	return nil
}
