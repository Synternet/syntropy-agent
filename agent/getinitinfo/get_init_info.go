package getinitinfo

import (
	"encoding/json"
	"io"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/docker"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const (
	cmd     = "GET_INIT_INFO"
	cmdResp = "INIT_INFO"
	pkgName = "GetInitInfo. "
)

type getInfoRequest struct {
	common.MessageHeader
	Data interface{} `json:"data,omitempty"`
}

type InitInfoAgent struct {
	Provider          uint     `json:"provider,omitempty"` // 0 is not used and do not send
	Tags              []string `json:"tags"`
	LocationLatitude  float32  `json:"location_lat,omitempty"`
	LocationLongitude float32  `json:"location_lon,omitempty"`
}

type InitInfoConfig struct {
	Status bool `json:"auto_connect_services"`
}

type getInfoResponse struct {
	common.MessageHeader
	Data struct {
		Agent  InitInfoAgent  `json:"agent,omitempty"`
		Config InitInfoConfig `json:"config,omitempty"`
	} `json:"data"`
}

type getInitInfo struct {
	w      io.Writer
	docker docker.DockerHelper
}

func New(w io.Writer, d docker.DockerHelper) common.Command {
	return &getInitInfo{
		w:      w,
		docker: d,
	}
}

func (obj *getInitInfo) Name() string {
	return cmd
}

func (obj *getInitInfo) Exec(raw []byte) error {
	var req getInfoRequest
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	resp := getInfoResponse{
		MessageHeader: req.MessageHeader,
	}
	resp.MsgType = cmdResp

	resp.Data.Agent.Provider = config.GetAgentProvider()
	resp.Data.Agent.Tags = config.GetAgentTags()
	resp.Data.Agent.LocationLatitude = config.GetLocationLatitude()
	resp.Data.Agent.LocationLongitude = config.GetLocationLongitude()
	resp.Data.Config.Status = config.GetServicesStatus()

	arr, err := json.Marshal(&resp)
	if err != nil {
		return err
	}

	logger.Debug().Println(pkgName, "Sending: ", string(arr))
	obj.w.Write(arr)
	return err
}
