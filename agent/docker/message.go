package docker

import "github.com/SyntropyNet/syntropy-agent-go/pkg/common"

type networkInfoMessage struct {
	common.MessageHeader
	Data []DockerNetworkInfoEntry `json:"data"`
}

type containerInfoMessage struct {
	common.MessageHeader
	Data []DockerContainerInfoEntry `json:"data"`
}

type DockerNetworkInfoEntry struct {
	ID      string   `json:"agent_network_id"`
	Name    string   `json:"agent_network_name"`
	Subnets []string `json:"agent_network_subnets"`
}

type DockerContainerInfoEntry struct {
	ID       string   `json:"agent_container_id"`
	Name     string   `json:"agent_container_name"`
	State    string   `json:"agent_container_state"`
	Uptime   string   `json:"agent_container_uptime"`
	Networks []string `json:"agent_container_networks"`
	IPs      []string `json:"agent_container_ips"`
	Ports    struct {
		TCP []int `json:"tcp"`
		UDP []int `json:"udp"`
	} `json:"agent_container_ports"`
}
