package wgconf

import (
	"fmt"
	"net/netip"
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

func (e *wgConfEntry) asPeerInfo() (*swireguard.PeerInfo, error) {
	var ifname string
	if strings.HasPrefix(e.Args.IfName, env.InterfaceNamePrefix) {
		ifname = e.Args.IfName
	} else {
		ifname = env.InterfaceNamePrefix + e.Args.IfName
	}

	pi := &swireguard.PeerInfo{
		IfName:       ifname,
		PublicKey:    e.Args.PublicKey,
		ConnectionID: e.Metadata.ConnectionID,
		GroupID:      e.Metadata.GroupID,
		AgentID:      e.Metadata.AgentID,
		Port:         e.Args.EndpointPort,
	}

	// These values may be absent on peer delete messages. Ignore errors.
	// Don't worry about values - they will be taken from cache
	pi.IP, _ = netip.ParseAddr(e.Args.EndpointIPv4)
	pi.Gateway, _ = netip.ParseAddr(e.Args.GatewayIPv4)

	for _, ipStr := range e.Args.AllowedIPs {
		aip, err := netip.ParsePrefix(ipStr)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed IP %s: %s", ipStr, err)
		}
		pi.AllowedIPs = append(pi.AllowedIPs, aip)
	}

	return pi, nil
}

func (e *wgConfEntry) asNetworkPath() (*common.SdnNetworkPath, error) {
	if len(e.Args.AllowedIPs) == 0 {
		return nil, fmt.Errorf("no IP address is present")
	}

	netpath := &common.SdnNetworkPath{
		Ifname:       e.Args.IfName,
		PublicKey:    e.Args.PublicKey,
		ConnectionID: e.Metadata.ConnectionID,
		GroupID:      e.Metadata.GroupID,
	}

	// Controller sends first IP as connected peers internal IP address
	// Use this IP as internal routing gateway
	ip, err := netip.ParsePrefix(e.Args.AllowedIPs[0])
	if err != nil {
		return nil, fmt.Errorf("failed parsing IP address %s: %s", e.Args.AllowedIPs[0], err)
	}
	netpath.Gateway = ip.Addr()

	return netpath, nil
}

type wgConfMsg struct {
	common.MessageHeader
	Data []wgConfEntry `json:"data"`
}
