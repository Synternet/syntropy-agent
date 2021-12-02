package wgconf

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/mole"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const (
	pkgName = "Wg_Conf. "
	cmd     = "WG_CONF"
)

type wgConf struct {
	writer io.Writer
	mole   *mole.Mole
}

func New(w io.Writer, m *mole.Mole) common.Command {
	return &wgConf{
		writer: w,
		mole:   m,
	}
}

func (obj *wgConf) Name() string {
	return cmd
}

func (obj *wgConf) Exec(raw []byte) error {
	var req wgConfMsg
	var errorCount int
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	for _, cmd := range req.Data {
		switch cmd.Function {
		case "add_peer":
			wgp := cmd.asPeerInfo()
			err = obj.mole.AddPeer(wgp, &common.SdnNetworkPath{
				Ifname:       cmd.Args.IfName,
				Gateway:      cmd.Args.GatewayIPv4,
				ConnectionID: cmd.Metadata.ConnectionID,
				GroupID:      cmd.Metadata.GroupID,
			})

		case "remove_peer":
			wgp := cmd.asPeerInfo()
			err = obj.mole.RemovePeer(wgp, &common.SdnNetworkPath{
				Ifname: cmd.Args.IfName,
			})
		}

		if err != nil {
			errorCount++
			logger.Error().Println(pkgName, cmd.Function, err)
		}

	}

	if errorCount > 0 {
		errResp := common.ErrorResponse{
			MessageHeader: req.MessageHeader,
		}
		errResp.Data.Type = cmd + "_ERROR"
		errResp.Data.Message = fmt.Sprintf("There were %d errors while performing %s request %s",
			errorCount, req.MsgType, req.ID)
		errResp.Now()
		arr, err := json.Marshal(errResp)
		if err != nil {
			return err
		}
		// Tricky here: I have errors, and I send them back to controller
		// But they are not internal application errors
		logger.Debug().Println(pkgName, "Sending: ", string(raw))
		obj.writer.Write(arr)
		return nil
	}

	// sync and merge everything between controller and OS
	obj.mole.Apply()

	return nil
}
