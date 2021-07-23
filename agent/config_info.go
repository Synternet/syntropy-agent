package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/SyntropyNet/syntropy-agent-go/config"
	netiface "github.com/SyntropyNet/syntropy-agent-go/network/interface"
	"github.com/SyntropyNet/syntropy-agent-go/wireguard"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
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
		VPN []configInfoVpnEntry `json:"vpn,omitempty"`
	} `json:"data"`
}

func (req *configInfoMsg) HasInterface(ifname string) bool {
	fixedNames := []string{"PUBLIC"}

	for _, n := range fixedNames {
		if ifname == n {
			return true
		}
	}

	return false
}

type updateAgentConfigEntry struct {
	Function string `json:"fn"`
	Data     struct {
		IfName    string `json:"ifname"`
		PublicKey string `json:"public_key"`
		IP        string `json:"internal_ip,omitempty"`
		Port      int    `json:"listen_port,omitempty"`
	} `json:"data"`
}

type updateAgentConfigMsg struct {
	messageHeader
	Data []updateAgentConfigEntry `json:"data"`
}

func createInterface(a *Agent, ifname string, e *configInfoNetworkEntry) (*updateAgentConfigEntry, error) {
	var port int

	wgdevs, err := a.wg.Devices()
	if err != nil {
		log.Println("wgctrl.Devices: ", err)
		return nil, err
	}
	for _, w := range wgdevs {
		if ifname == w.Name {
			log.Println("Skipping existing interface ", ifname)
			return nil, nil
		}
	}

	if e == nil {
		return nil, fmt.Errorf("invalid parameters to createInterface")
	}
	if e.Port != 0 {
		port = e.Port
	} else {
		port = wireguard.GetFreePort(ifname)
	}

	log.Println("Creating interface ", ifname, e, port)
	err = netiface.CreateInterfaceCmd(ifname)
	if err != nil {
		return nil, fmt.Errorf("create wg interface failed: %s", err.Error())
	}

	privKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generate private key error: %s", err.Error())
	}

	cfg := wgtypes.Config{
		PrivateKey: &privKey,
		ListenPort: &port,
	}
	err = a.wg.ConfigureDevice(ifname, cfg)
	if err != nil {
		return nil, fmt.Errorf("configure interface %s failed: %s", ifname, err.Error())
	}

	netiface.SetInterfaceUpCmd(ifname)
	netiface.SetInterfaceIPCmd(ifname, e.IP)

	dev, err := a.wg.Device(ifname)
	if err != nil {
		return nil, fmt.Errorf("get device %s failed: %s", ifname, err.Error())
	}

	rv := &updateAgentConfigEntry{
		Function: "create_interface",
	}
	rv.Data.IfName = ifname
	rv.Data.IP = e.IP
	rv.Data.Port = dev.ListenPort
	rv.Data.PublicKey = dev.PublicKey.String()

	return rv, nil
}

func configInfo(a *Agent, raw []byte) (rv []byte, err error) {
	var req configInfoMsg
	err = json.Unmarshal(raw, &req)
	if err != nil {
		return nil, err
	}
	log.Println(req)
	resp := updateAgentConfigMsg{
		messageHeader: req.messageHeader,
	}
	log.Print("Initial responce: ", resp)
	resp.MsgType = "UPDATE_AGENT_CONFIG"

	// Dump pretty idented json to temp file
	prettyJson, err := json.MarshalIndent(req, "", "    ")
	if err != nil {
		return nil, err
	}
	os.WriteFile(config.AgentTempDir+"/config_dump", prettyJson, 0600)

	wgdevs, err := a.wg.Devices()
	if err != nil {
		log.Println("wgctrl.Devices: ", err)
		return nil, err
	}
	log.Println("Existing wireguard interfaces: ", wgdevs)

	// create missing interfaces
	respEntry, err := createInterface(a, "SYNTROPY_PUBLIC", &req.Data.Network.Public)
	if err != nil {
		return nil, err
	}
	resp.Data = append(resp.Data, *respEntry)
	respEntry, err = createInterface(a, "SYNTROPY_SDN1", &req.Data.Network.Sdn1)
	if err != nil {
		return nil, err
	}
	resp.Data = append(resp.Data, *respEntry)
	respEntry, err = createInterface(a, "SYNTROPY_SDN2", &req.Data.Network.Sdn2)
	if err != nil {
		return nil, err
	}
	resp.Data = append(resp.Data, *respEntry)
	respEntry, err = createInterface(a, "SYNTROPY_SDN3", &req.Data.Network.Sdn3)
	if err != nil {
		return nil, err
	}
	resp.Data = append(resp.Data, *respEntry)

	wgdevs, err = a.wg.Devices()
	if err != nil {
		log.Println("wgctrl.Devices: ", err)
		return nil, err
	}
	log.Println("Existing/created wireguard interfaces: ", wgdevs)

	resp.Now()

	arr, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	a.Transmit(arr)

	return nil, nil
}
