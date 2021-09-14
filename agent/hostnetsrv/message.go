package hostnetsrv

import "github.com/SyntropyNet/syntropy-agent-go/pkg/common"

type hostNetworkServicesMessage struct {
	common.MessageHeader
	Data []hostServiceEntry `json:"data"`
}

type hostServiceEntry struct {
	IfName  string   `json:"agent_network_iface"`
	Subnets []string `json:"agent_network_subnets"`
	Ports   ports    `json:"agent_network_ports"`
}

type ports struct {
	TCP []int `json:"tcp"`
	UDP []int `json:"udp"`
}
