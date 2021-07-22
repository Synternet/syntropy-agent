package agent

import (
	"encoding/json"
	"log"
)

type configInfoNetworkEntry struct {
	IP        string `json:"internal_ip"`
	PublicKey string `json:"public_key,omitempty"`
	Port      int    `json:"listen_port"`
}

/****    TODO: review me      ******/
//	I'm not sure this is a good idea, but I wanted to decode json in one step
//	So I am mixing different structs in one instance
//	And will try to use only correct fields, depending on `fn` type
type configInfoVpnEntry struct {
	Function string `json:"fn"`

	Args struct {
		// Common fields
		IfName string `json:"ifname"`
		// create_interface
		InternalIP string `json:"internal_ip,omitempty"`
		// add_peer
		AllowedIPs   []string `json:"allowed_ips,omitempty"`
		EndpointIPv4 string   `json:"endpoint_ipv4,omitempty"`
		EndpointPort int      `json:"endpoint_port,omitempty"`
		PublicKey    string   `json:"public_key,omitempty"`
		GatewayIPv4  string   `json:"gw_ipv4,omitempty"`
	} `json:"args,omitempty"`

	Metadata struct {
		// create_interface
		NetworkID int `json:"network_id,omitempty"`
		// add_peer
		DeviceID         string `json:"device_id,omitempty"`
		DeviceName       string `json:"device_name,omitempty"`
		DevicePublicIPv4 string `json:"device_public_ipv4,omitempty"`
		ConnectionID     int    `json:"connection_id,omitempty"`
	} `json:"metadata,omitempty"`
}

type configInfoMsg struct {
	messageHeader
	Data struct {
		AgentID int `json:"agent_id"`
		Network struct {
			Public configInfoNetworkEntry `json:"PUBLIC"`
			Sdn1   configInfoNetworkEntry `json:"SDN1"`
			Sdn2   configInfoNetworkEntry `json:"SDN2"`
			Sdn3   configInfoNetworkEntry `json:"SDN3"`
		}
	} `json:"data"`
	VPN []configInfoVpnEntry `json:"vpn,omitempty"`
}

func configInfo(a *Agent, raw []byte) (rv []byte, err error) {
	var cfg configInfoMsg
	err = json.Unmarshal(raw, &cfg)
	if err != nil {
		return nil, err
	}

	log.Println(cfg)

	return nil, nil
}
