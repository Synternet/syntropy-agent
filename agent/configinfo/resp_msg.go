package configinfo

import (
	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
)

type updateAgentConfigEntry struct {
	Function string `json:"fn"`
	Data     struct {
		IfName    string `json:"ifname"`
		PublicKey string `json:"public_key"`
		IP        string `json:"internal_ip"`
		Port      int    `json:"listen_port"`
	} `json:"data"`
}

type updateAgentConfigMsg struct {
	common.MessageHeader
	Data []updateAgentConfigEntry `json:"data"`
}

func (msg *updateAgentConfigMsg) AddInterface(data *swireguard.InterfaceInfo) {
	e := updateAgentConfigEntry{Function: "create_interface"}
	e.Data.IfName = data.IfName
	e.Data.IP = data.IP
	e.Data.PublicKey = data.PublicKey
	e.Data.Port = data.Port

	msg.Data = append(msg.Data, e)
}

func (msg *updateAgentConfigMsg) AddPeer(data *swireguard.PeerInfo) {
	e := updateAgentConfigEntry{Function: "add_peer"}
	e.Data.IfName = data.IfName
	e.Data.IP = data.IP
	e.Data.PublicKey = data.PublicKey
	e.Data.Port = data.Port

	msg.Data = append(msg.Data, e)
}
