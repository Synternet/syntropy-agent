package setconfig

import (
	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
)

type interfaceEntry struct {
	IfName    string `json:"ifname"`
	PublicKey string `json:"public_key"`
	IP        string `json:"internal_ip"`
	Port      int    `json:"listen_port"`
}

type configInfoEntry struct {
	Interfaces []interfaceEntry `json:"interfaces"`
}

type ConfigInfoMsg struct {
	common.MessageHeader
	Data configInfoEntry `json:"data"`
}

func (msg *ConfigInfoMsg) AddInterface(data *swireguard.InterfaceInfo) {
	e := interfaceEntry{}
	e.IfName = data.IfName
	e.IP = data.IP.String()
	e.PublicKey = data.PublicKey
	e.Port = data.Port
	msg.Data.Interfaces = append(msg.Data.Interfaces, e)
}

//func (msg *updateAgentConfigMsg) AddPeer(data *swireguard.PeerInfo) {
//	e := configInfoEntry{Function: "add_peer"}
//	e.Data.IfName = data.IfName
//	e.Data.IP = data.IP.String()
//	e.Data.PublicKey = data.PublicKey
//	e.Data.Port = data.Port
//
//	msg.Data = append(msg.Data, e)
//}
