package kubernet

import "github.com/SyntropyNet/syntropy-agent-go/pkg/common"

type kubernetesInfoMessage struct {
	common.MessageHeader
	Data []kubernetesServiceEntry `json:"data"`
}

type kubernetesServiceEntry struct {
	Name    string   `json:"agent_service_name"`
	Subnets []string `json:"agent_service_subnets"`
	Ports   ports    `json:"agent_service_ports"`
}

type ports struct {
	TCP []int `json:"tcp"`
	UDP []int `json:"udp"`
}
