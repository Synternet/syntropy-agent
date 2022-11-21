package setconfig

import (
	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/agent/swireguard"
)

type InterfaceEntry struct {
	Index     int    `json:"index"`
	PublicKey string `json:"public_key"`
	IP        string `json:"internal_ip"`
	Port      int    `json:"port"`
}

type ConfigInfoEntry struct {
	Interfaces []InterfaceEntry `json:"interfaces"`
}

type ConfigInfoMsg struct {
	common.MessageHeader
	Data ConfigInfoEntry `json:"data"`
}

func (msg *ConfigInfoMsg) AddInterface(data *swireguard.InterfaceInfo) {
	e := InterfaceEntry{}
	e.Index = data.IfIndex
	e.IP = data.IP.String()
	e.PublicKey = data.PublicKey
	e.Port = data.Port
	msg.Data.Interfaces = append(msg.Data.Interfaces, e)
}
