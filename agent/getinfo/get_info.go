package getinfo

import (
	"encoding/json"
	"io"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/agent/docker"
	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/pubip"
)

const (
	cmd     = "GET_INFO"
	pkgName = "GetInfo. "
)

type getInfoRequest struct {
	common.MessageHeader
	Data interface{} `json:"data,omitempty"`
}

type getInfoResponse struct {
	common.MessageHeader
	Data struct {
		Provider          int      `json:"agent_provider,omitempty"` // 0 is not used and do not send
		Status            bool     `json:"service_status"`
		Tags              []string `json:"agent_tags"`
		ExternalIP        string   `json:"external_ip"`
		LocationLatitude  float32  `json:"location_lat,omitempty"`
		LocationLongitude float32  `json:"location_lon,omitempty"`

		NetworkInfo   []docker.DockerNetworkInfoEntry   `json:"network_info"`
		ContainerInfo []docker.DockerContainerInfoEntry `json:"container_info"`
	} `json:"data"`
}

type getInfo struct {
	w      io.Writer
	docker docker.DockerHelper
}

func New(w io.Writer, d docker.DockerHelper) common.Command {
	return &getInfo{
		w:      w,
		docker: d,
	}
}

func (obj *getInfo) Name() string {
	return cmd
}

func (obj *getInfo) Exec(raw []byte) error {
	var req getInfoRequest
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	resp := getInfoResponse{
		MessageHeader: req.MessageHeader,
	}

	resp.Data.Provider = config.GetAgentProvider()
	resp.Data.Status = config.GetServicesStatus()
	resp.Data.Tags = config.GetAgentTags()
	resp.Data.ExternalIP = pubip.GetPublicIp().String()
	resp.Data.LocationLatitude = config.GetLocationLatitude()
	resp.Data.LocationLongitude = config.GetLocationLongitude()
	resp.Data.NetworkInfo = obj.docker.NetworkInfo()
	resp.Data.ContainerInfo = obj.docker.ContainerInfo()

	arr, err := json.Marshal(&resp)
	if err != nil {
		return err
	}

	logger.Debug().Println(pkgName, "Sending: ", string(arr))
	obj.w.Write(arr)

	return err
}
