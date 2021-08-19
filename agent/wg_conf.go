package agent

import (
	"encoding/json"

	"github.com/SyntropyNet/syntropy-agent-go/logger"
	"github.com/SyntropyNet/syntropy-agent-go/wireguard"
)

type allowedIPsInfoEntry struct {
	ServiceName string `json:"agent_service_name,omitempty"`
	TcpPorts    []int  `json:"agent_service_tcp_ports,omitempty"`
	UdpPorts    []int  `json:"agent_service_udp_ports,omitempty"`
	SubnetIP    string `json:"agent_service_subnet_ip,omitempty"`
}

type wgConfEntry struct {
	Function string `json:"fn"`
	Args     struct {
		// Interface configuration
		IfName string `json:"ifname,omitempty"`
		IP     string `json:"internal_ip,omitempty"`
		Port   int    `json:"listen_port,omitempty"`
		// Peer configuration
		PublicKey    string   `json:"public_key,omitempty"`
		AllowedIPs   []string `json:"allowed_ips,omitempty"`
		GatewayIPv4  string   `json:"gw_ipv4,omitempty"`
		EndpointIPv4 string   `json:"endpoint_ipv4,omitempty"`
		EndpointPort int      `json:"endpoint_port,omitempty"`
	}
	Metadata struct {
		// Interface configuration
		NetworkID int `json:"network_id,omitempty"`
		// Peer configuration
		DeviceID         string                `json:"device_id,omitempty"`
		DeviceName       string                `json:"device_name,omitempty"`
		DevicePublicIPv4 string                `json:"device_public_ipv4,omitempty"`
		ConnectionID     int                   `json:"connection_id,omitempty"`
		AllowedIPsInfo   []allowedIPsInfoEntry `json:"allowed_ips_info,omitempty"`
	}
}

func (e *wgConfEntry) AsPeerInfo() *wireguard.PeerInfo {
	return &wireguard.PeerInfo{
		IfName:     e.Args.IfName,
		IP:         e.Args.EndpointIPv4,
		PublicKey:  e.Args.PublicKey,
		Port:       e.Args.EndpointPort,
		Gateway:    e.Args.GatewayIPv4,
		AllowedIPs: e.Args.AllowedIPs,
	}
}

func (e *wgConfEntry) AsInterfaceInfo() *wireguard.InterfaceInfo {
	return &wireguard.InterfaceInfo{
		IfName:    e.Args.IfName,
		IP:        e.Args.IP,
		PublicKey: e.Args.PublicKey,
		Port:      e.Args.Port,
	}
}

type wgConfReq struct {
	messageHeader
	Data []wgConfEntry `json:"data"`
}

func wireguardConfigure(a *Agent, raw []byte) error {
	var req wgConfReq
	var errorCount int
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	for _, cmd := range req.Data {
		switch cmd.Function {
		case "add_peer":
			err = a.wg.AddPeer(cmd.AsPeerInfo())

		case "remove_peer":
			err = a.wg.RemovePeer(cmd.AsPeerInfo())

		case "create_interface":
			wgi := cmd.AsInterfaceInfo()
			err = a.wg.CreateInterface(wgi)
			/*
				if err == nil &&
					cmd.Args.PublicKey != wgi.PublicKey ||
					cmd.Args.ListenPort != wgi.Port {
					resp.AddInterface(wgi)
				}
			*/

		case "remove_interface":
			wgi := cmd.AsInterfaceInfo()
			err = a.wg.RemoveInterface(wgi)
		}
		if err != nil {
			errorCount++
			logger.Error().Println(pkgName, cmd.Function, err)
		}

	}

	if errorCount > 0 {
		// TODO: add sending errors to controller
	}

	// TODO: send back ACTUAL info (e.g. ports may change, or create_interface public key)
	req.Now()
	respArr, err := json.Marshal(req)
	if err != nil {
		return err
	}
	a.Write(respArr)
	return nil
}
