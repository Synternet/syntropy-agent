package supportinfo

import (
	"bytes"
	"encoding/json"
	"io"
	"os/exec"

	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
)

const (
	cmd     = "SUPPORT_INFO_CTA"
	cmdResp = "SUPPORT_INFO_ATC"
)

type supportInfoRequest struct {
	common.MessageHeader
	Data interface{} `json:"data,omitempty"`
}

type SupportInfoEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type supportInfoResponse struct {
	common.MessageHeader
	Data []*SupportInfoEntry `json:"data"`
}

type supportInfo struct {
	w io.Writer
}

func New(w io.Writer) common.Command {
	return &supportInfo{
		w: w,
	}
}

func (obj *supportInfo) Name() string {
	return cmd
}

func GetSupportInfoEntries() []*SupportInfoEntry {
	entries := make([]*SupportInfoEntry, 2)
	// wg_info msg
	entries = append(entries,
		&SupportInfoEntry{
			Key:   "wg_info",
			Value: fetchCmdExecOutput("wg", "show"),
		})
	// routes msg
	entries = append(entries,
		&SupportInfoEntry{
			Key:   "routes",
			Value: fetchCmdExecOutput("route", "-n"),
		})
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
	resp.Data = GetSupportInfoEntries()

	arr, err := json.Marshal(&resp)
	if err != nil {
		return err
	}

	obj.w.Write(arr)

	return err
}

func fetchCmdExecOutput(cmdName string, arg string) string {
	cmd := exec.Command(cmdName, arg)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		return errb.String()
	}
	return outb.String()
}
