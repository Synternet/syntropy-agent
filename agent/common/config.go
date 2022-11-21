package common

import (
	"net/netip"
	"strconv"
	"strings"

	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
)

type ConfigInterfaceEntry struct {
	Index     int    `json:"index"`
	IP        string `json:"internal_ip"`
	PublicKey string `json:"public_key,omitempty"`
	Port      int    `json:"listen_port,omitempty"`
}

func (e *ConfigInterfaceEntry) asInterfaceInfo(IfIndex int) (*swireguard.InterfaceInfo, error) {
	var ifname string
	ifaceIndexStr := strconv.Itoa(IfIndex)
	if strings.HasPrefix(ifaceIndexStr, env.InterfaceNamePrefix) {
		ifname = strconv.Itoa(IfIndex)
	} else {
		ifname = env.InterfaceNamePrefix + ifaceIndexStr
	}

	addr, err := netip.ParseAddr(e.IP)
	if err != nil {
		return nil, err
	}

	return &swireguard.InterfaceInfo{
		IfName:    ifname,
		IfIndex:   IfIndex,
		IP:        addr,
		PublicKey: e.PublicKey,
		Port:      e.Port,
	}, nil
}

func (e *configServicesEntry) AsServiceInfo() (*swireguard.ServiceInfo, error) {

	pi := &swireguard.ServiceInfo{
		ConnectionIDs: e.ConnectionIDs,
	}

	pi.IP, _ = netip.ParsePrefix(e.IP)

	return pi, nil
}

func (e *configPeersEntry) AsPeerInfo() (*swireguard.PeerInfo, error) {
	var ifname string

	ifname = env.InterfaceNamePrefix + strconv.Itoa(e.Index)

	pi := &swireguard.PeerInfo{
		IfName:       ifname,
		IfIndex:      e.Index,
		PublicKey:    e.PublicKey,
		ConnectionID: e.ConnectionID,
		GroupID:      e.GroupID,
		AgentID:      e.AgentID,
		Port:         e.Port,
	}

	// These values may be absent on peer delete messages. Ignore errors.
	// Don't worry about values - they will be taken from cache
	pi.IP, _ = netip.ParseAddr(e.PublicIP)

	aip, _ := netip.ParsePrefix(e.PrivateIP + "/32")
	pi.AllowedIPs = append(pi.AllowedIPs, aip)

	return pi, nil
}

func (e *ConfigInterfaceEntry) AsInterfaceInfo() (*swireguard.InterfaceInfo, error) {

	ifname := env.InterfaceNamePrefix + strconv.Itoa(e.Index)

	addr, err := netip.ParseAddr(e.IP)
	if err != nil {
		return nil, err
	}
	return &swireguard.InterfaceInfo{
		IfName:    ifname,
		IfIndex:   e.Index,
		IP:        addr,
		PublicKey: e.PublicKey,
		Port:      e.Port,
	}, nil
}

func (e *configPeersEntry) AsNetworkPath() (*SdnNetworkPath, error) {
	ifname := env.InterfaceNamePrefix + strconv.Itoa(e.Index)
	netpath := &SdnNetworkPath{
		Ifname:       ifname,
		PublicKey:    e.PublicKey,
		ConnectionID: e.ConnectionID,
		GroupID:      e.GroupID,
	}

	// Use this IP as internal routing gateway
	netpath.Gateway, _ = netip.ParseAddr(e.PrivateIP)

	return netpath, nil
}

type configInfoSubnetworksEntry struct {
	Name   string `json:"name"`
	Subnet string `json:"subnet"`
	Type   string `json:"type"`
}

type configPeersEntry struct {
	Action       string `json:"action"`
	Index        int    `json:"index"`
	PublicKey    string `json:"public_key,omitempty"`
	PrivateIP    string `json:"private_ip,omitempty"`
	PublicIP     string `json:"public_ip,omitempty"`
	Port         int    `json:"port,omitempty"`
	ConnectionID int    `json:"connection_id,omitempty"`
	GroupID      int    `json:"connection_group_id,omitempty"`
	AgentID      int    `json:"agent_id,omitempty"`
}

type configServicesEntry struct {
	Action        string `json:"action"`
	IP            string `json:"ip"`
	Name          string `json:"name"`
	ConnectionIDs []int  `json:"connection_ids,omitempty"`
	Ports         Ports  `json:"ports"`
}

type configSettingsReroutingEntry struct {
	Action             string  `json:"action"`
	LatencyCoefficient float32 `json:"latency_coefficient"`
	ReroutingThreshold float32 `json:"rerouting_threshold"`
	GroupIDs           []int   `json:"connection_group_ids,omitempty"`
}

type ConfigSettingsAutopingEntry struct {
	MessageHeader
	IPs       []string `json:"ips"`
	Interval  int      `json:"interval"`
	RespLimit int      `json:"response_limit"`
}

type configSettingsEntry struct {
	Rerouting []configSettingsReroutingEntry `json:"rerouting"`
	Autoping  ConfigSettingsAutopingEntry    `json:"auto_ping"`
}

type ConfigMsg struct {
	MessageHeader
	Data struct {
		AgentID     int                          `json:"agent_id"`
		Interfaces  []ConfigInterfaceEntry       `json:"interfaces,omitempty"`
		Peers       []configPeersEntry           `json:"peers,omitempty"`
		Services    []configServicesEntry        `json:"services,omitempty"`
		Subnetworks []configInfoSubnetworksEntry `json:"subnetworks,omitempty"`
		Settings    configSettingsEntry          `json:"settings,omitempty"`
	} `json:"data"`
}
