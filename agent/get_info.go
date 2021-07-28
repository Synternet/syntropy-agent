package agent

import (
	"encoding/json"
	"log"

	"github.com/SyntropyNet/syntropy-agent-go/config"
)

type getInfoRequest struct {
	messageHeader
	Data interface{} `json:"data,omitempty"`
}

type NetworkInfoEntry struct {
	ID      string   `json:"agent_network_id,omitempty"`
	Name    string   `json:"agent_network_name,omitempty"`
	Subnets []string `json:"agent_network_subnets,omitempty"`
	// TODO: review why different names in array ?
	/*
		""network_info?"": [
			{
				""agent_network_id"": ""agent_network_id"",
				""agent_network_name"": ""agent_network_name"",
				""agent_network_subnets"": [
					""1.2.3.4/12""
				]
			},
			{
				""docker_network_id"": ""agent_network_id"",
				""docker_network_name"": ""agent_network_name"",
				""docker_network_subnets"": [
					""1.2.3.4/12""
				]
			}
		]
	*/
}

type ContainerInfoEntry struct {
	ID   string `json:"agent_container_id,omitempty"`
	Name string `json:"agent_container_name,omitempty"`

	// TODO: review for complete structure
	/*
		""container_info"":  [
				{
					""agent_container_id"": ""4e67bdb06bb2a9e19a61ad5a420b8701115263fe56b2918547cc9138084bf1c9"",
					""agent_container_name"": ""pgadmin"",
					""agent_container_networks: [aaa,bbb],
					""agent_container_ips"": [""172.18.0.2""],
					""agent_container_ports"": {""udp"": [], ""tcp"": [443, 5050, 5050, 80]},
					""agent_container_state"": ""running"",
					""agent_container_uptime"": ""Up About a minute""
				},
				{
					""agent_container_id"": ""5d1774cb76c9385dcd025abbc84faea12dc7d7f247597042882361a7baa86fe6"",
					""agent_container_name"": ""postgres"",
					""agent_container_ips: ['aaa', 'bbb'],
					""agent_container_subnets"": [""172.18.0.3/16""],
					""agent_container_ports"": {
							""udp"": [],
							""tcp"": [5432, 5435]
					},
					""agent_container_state"": ""running"",
					""agent_container_uptime"": ""Up About a minute""
				}
		]
	*/
}

type getInfoResponce struct {
	messageHeader
	Data struct {
		Provider   int      `json:"agent_provider,omitempty"` // 0 is not used and do not send
		Status     bool     `json:"service_status"`
		Tags       []string `json:"agent_tags"`
		ExternalIP string   `json:"external_ip"`

		NetworkInfo   []NetworkInfoEntry   `json:"network_info"`
		ContainerInfo []ContainerInfoEntry `json:"container_info"`
	} `json:"data"`
}

func getInfo(a *Agent, raw []byte) error {

	var req getInfoRequest
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	resp := getInfoResponce{
		messageHeader: req.messageHeader,
	}
	resp.Data.Provider = config.GetAgentProvider()
	resp.Data.Status = config.GetServicesStatus()
	resp.Data.Tags = config.GetAgentTags()
	resp.Data.ExternalIP = config.GetPublicIp()
	resp.Data.NetworkInfo = FakeNetworkInfo()
	resp.Data.ContainerInfo = FakeContainerInfo()

	arr, err := json.Marshal(&resp)
	if err != nil {
		log.Println("Marshal error: ", err)
		return err
	}

	a.Write(arr)

	return err
}
