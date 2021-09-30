package hostnetsrv

import "github.com/SyntropyNet/syntropy-agent-go/pkg/common"

type hostNetworkServicesMessage struct {
	common.MessageHeader
	Data []hostServiceEntry `json:"data"`
}

type hostServiceEntry struct {
	Name    string       `json:"agent_network_iface"`
	Subnets []string     `json:"agent_network_subnets"`
	Ports   common.Ports `json:"agent_network_ports"`
}
