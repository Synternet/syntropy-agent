package wgconf

import (
	"strings"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
)

// This struct is not used in Linux agent
// Is is intendended only for desktop (MacOS and Windows) application
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
		GroupID          int                   `json:"connection_group_id,omitempty"`
		AgentID          int                   `json:"agent_id,omitempty"`
		AllowedIPsInfo   []allowedIPsInfoEntry `json:"allowed_ips_info,omitempty"`
	}
}

func (e *wgConfEntry) asPeerInfo() *swireguard.PeerInfo {
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

func (e *wgConfEntry) asNetworkPath() *common.SdnNetworkPath {
	return &common.SdnNetworkPath{
		Ifname:       e.Args.IfName,
		PublicKey:    e.Args.PublicKey,
		Gateway:      e.Args.GatewayIPv4,
		ConnectionID: e.Metadata.ConnectionID,
		GroupID:      e.Metadata.GroupID,
	}
}

type wgConfMsg struct {
	common.MessageHeader
	Data []wgConfEntry `json:"data"`
}
