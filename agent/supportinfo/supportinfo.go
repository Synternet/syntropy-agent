package supportinfo

import (
	"encoding/json"
	"io"

	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
)

const (
	cmd     = "SUPPORT_INFO_CTA"
	cmdResp = "SUPPORT_INFO_ATC"
	pkgName = "SupportInfo. "
)

type supportInfoRequest struct {
	common.MessageHeader
	Data interface{} `json:"data,omitempty"`
}

type supportInfoResponse struct {
	common.MessageHeader
	Data []*common.KeyValue `json:"data"`
}

type supportInfo struct {
	w       io.Writer
	helpers []common.SupportInfoHelper
}

func New(w io.Writer, sihelper ...common.SupportInfoHelper) common.Command {
	return &supportInfo{
		w:       w,
		helpers: sihelper,
	}
}

func (obj *supportInfo) Name() string {
	return cmd
}

func (obj *supportInfo) getSupportInfoEntries() []*common.KeyValue {
	var entries []*common.KeyValue

	for _, helper := range obj.helpers {
		entries = append(entries, helper.SupportInfo())
	}

	return entries
}

func (obj *supportInfo) Exec(raw []byte) error {
	var req supportInfoRequest
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}
	resp := supportInfoResponse{
		MessageHeader: req.MessageHeader,
	}
	resp.MsgType = cmdResp
	resp.Data = obj.getSupportInfoEntries()

	arr, err := json.Marshal(&resp)
	if err != nil {
		return err
	}
	logger.Debug().Println(pkgName, "Sending: ", string(arr))
	obj.w.Write(arr)

	return err
}
