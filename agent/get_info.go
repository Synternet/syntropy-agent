package agent

import (
	"encoding/json"

	"github.com/SyntropyNet/syntropy-agent-go/config"
)

type getInfoRequest struct {
	messageHeader
	Data interface{} `json:"data,omitempty"`
}

type getInfoResponce struct {
	messageHeader
	Data struct {
		Provider   int      `json:"agent_provider,omitempty"` // 0 is not used and do not send
		Status     bool     `json:"service_status"`
		Tags       []string `json:"agent_tags"`
		ExternalIP string   `json:"external_ip"`

		NetworkInfo   []config.DockerNetworkInfoEntry   `json:"network_info"`
		ContainerInfo []config.DockerContainerInfoEntry `json:"container_info"`
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
	resp.Data.NetworkInfo = config.GetDockerNetworkInfo()
	resp.Data.ContainerInfo = config.GetDockerContainerInfo()

	arr, err := json.Marshal(&resp)
	if err != nil {
		return err
	}

	a.Write(arr)

	return err
}
