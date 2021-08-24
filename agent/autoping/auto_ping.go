// autoping package implement both: controller.Command and controller.Service
package autoping

import (
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/SyntropyNet/syntropy-agent-go/logger"
	"github.com/SyntropyNet/syntropy-agent-go/multiping"
)

const cmd = "AUTO_PING"
const pkgName = "Auto_Ping. "

type AutoPing struct {
	writer io.Writer
	ping   *multiping.MultiPing
}

type autoPingRequest struct {
	controller.MessageHeader
	Data struct {
		IPs       []string `json:"ips"`
		Interval  int      `json:"interval"`
		RespLimit int      `json:"responce_limit"`
	} `json:"data"`
}

type autoPingResponce struct {
	controller.MessageHeader
	Data struct {
		Pings []multiping.PingResult `json:"pings"`
	} `json:"data"`
}

func New(w io.Writer) *AutoPing {
	ap := AutoPing{
		writer: w,
	}
	ap.ping = multiping.New(&ap)
	return &ap
}

func (obj *AutoPing) Name() string {
	return cmd
}

func (obj *AutoPing) Exec(raw []byte) error {

	var req autoPingRequest
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	obj.ping.Stop()
	obj.ping.Period = time.Duration(req.Data.Interval) * time.Second
	obj.ping.LimitCount = req.Data.RespLimit
	obj.ping.Flush()
	obj.ping.AddHost(req.Data.IPs...)
	obj.ping.Start()

	return nil
}

func (obj *AutoPing) ProcessPingResults(pr []multiping.PingResult) {
	var resp autoPingResponce
	resp.Data.Pings = pr
	resp.MsgType = cmd
	resp.ID = "ID." + strconv.FormatInt(time.Now().Unix(), 10)
	resp.Now()

	if len(resp.Data.Pings) > 0 {
		arr, err := json.Marshal(resp)
		if err != nil {
			logger.Error().Println(pkgName, "Process Ping Results: ", err)
			return
		}

		obj.writer.Write(arr)
	}
}

func (obj *AutoPing) Start() error {
	// TODO: add universal way for service locking
	obj.ping.Start()
	return nil
}

func (obj *AutoPing) Stop() error {
	obj.ping.Stop()
	return nil
}
