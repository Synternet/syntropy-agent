package wgconf

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/wireguard"
)

const (
	pkgName = "Wg_Conf. "
	cmd     = "WG_CONF"
)

type wgConf struct {
	writer io.Writer
	wg     *wireguard.Wireguard
}

type allowedIPsInfoEntry struct {
	ServiceName string `json:"agent_service_name,omitempty"`
	TcpPorts    []int  `json:"agent_service_tcp_ports,omitempty"`
	UdpPorts    []int  `json:"agent_service_udp_ports,omitempty"`
	SubnetIP    string `json:"agent_service_subnet_ip,omitempty"`
}

type wgConfEntry struct {
	Function string `json:"fn"`
	Args     struct {
		// Interface configuration
		IfName string `json:"ifname,omitempty"`
		IP     string `json:"internal_ip,omitempty"`
		Port   int    `json:"listen_port,omitempty"`
		// Peer configuration
		PublicKey    string   `json:"public_key,omitempty"`
		AllowedIPs   []string `json:"allowed_ips,omitempty"`
		GatewayIPv4  string   `json:"gw_ipv4,omitempty"`
		EndpointIPv4 string   `json:"endpoint_ipv4,omitempty"`
		EndpointPort int      `json:"endpoint_port,omitempty"`
	}
	Metadata struct {
		// Interface configuration
		NetworkID int `json:"network_id,omitempty"`
		// Peer configuration
		DeviceID         string                `json:"device_id,omitempty"`
		DeviceName       string                `json:"device_name,omitempty"`
		DevicePublicIPv4 string                `json:"device_public_ipv4,omitempty"`
		ConnectionID     int                   `json:"connection_id,omitempty"`
		AllowedIPsInfo   []allowedIPsInfoEntry `json:"allowed_ips_info,omitempty"`
	}
}

func (e *wgConfEntry) asPeerInfo() *wireguard.PeerInfo {
	var ifname string
	if strings.HasPrefix(e.Args.IfName, env.InterfaceNamePrefix) {
		ifname = e.Args.IfName
	} else {
		ifname = env.InterfaceNamePrefix + e.Args.IfName
	}

	return &wireguard.PeerInfo{
		IfName:     ifname,
		IP:         e.Args.EndpointIPv4,
		PublicKey:  e.Args.PublicKey,
		Port:       e.Args.EndpointPort,
		Gateway:    e.Args.GatewayIPv4,
		AllowedIPs: e.Args.AllowedIPs,
	}
}

func (e *wgConfEntry) asInterfaceInfo() *wireguard.InterfaceInfo {
	var ifname string
	if strings.HasPrefix(e.Args.IfName, env.InterfaceNamePrefix) {
		ifname = e.Args.IfName
	} else {
		ifname = env.InterfaceNamePrefix + e.Args.IfName
	}

	return &wireguard.InterfaceInfo{
		IfName:    ifname,
		IP:        e.Args.IP,
		PublicKey: e.Args.PublicKey,
		Port:      e.Args.Port,
	}
}

type wgConfMsg struct {
	controller.MessageHeader
	Data []wgConfEntry `json:"data"`
}

func (msg *wgConfMsg) AddPeerCmd(cmd string, pi *wireguard.PeerInfo) {
	e := wgConfEntry{
		Function: cmd,
	}
	e.Args.IfName = pi.IfName
	e.Args.EndpointIPv4 = pi.IP
	e.Args.PublicKey = pi.PublicKey
	e.Args.EndpointPort = pi.Port
	e.Args.GatewayIPv4 = pi.Gateway
	e.Args.AllowedIPs = pi.AllowedIPs

	msg.Data = append(msg.Data, e)
}

func (msg *wgConfMsg) AddInterfaceCmd(cmd string, ii *wireguard.InterfaceInfo) {
	e := wgConfEntry{
		Function: cmd,
	}
	e.Args.IfName = ii.IfName
	e.Args.IP = ii.IP
	e.Args.PublicKey = ii.PublicKey
	e.Args.Port = ii.Port

	msg.Data = append(msg.Data, e)
}

func New(w io.Writer, wg *wireguard.Wireguard) controller.Command {
	return &wgConf{
		writer: w,
		wg:     wg,
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

	resp := wgConfMsg{
		MessageHeader: req.MessageHeader,
		Data:          []wgConfEntry{},
	}
	for _, cmd := range req.Data {
		switch cmd.Function {
		case "add_peer":
			wgp := cmd.asPeerInfo()
			err = obj.wg.AddPeer(wgp)
			resp.AddPeerCmd(cmd.Function, wgp)

		case "remove_peer":
			wgp := cmd.asPeerInfo()
			err = obj.wg.RemovePeer(wgp)
			resp.AddPeerCmd(cmd.Function, wgp)

		case "create_interface":
			wgi := cmd.asInterfaceInfo()
			err = obj.wg.CreateInterface(wgi)
			resp.AddInterfaceCmd(cmd.Function, wgi)

		case "remove_interface":
			wgi := cmd.asInterfaceInfo()
			err = obj.wg.RemoveInterface(wgi)
			resp.AddInterfaceCmd(cmd.Function, wgi)
		}
		if err != nil {
			errorCount++
			logger.Error().Println(pkgName, cmd.Function, err)
		}

	}

	if errorCount > 0 {
		errResp := controller.ErrorResponce{
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
		obj.writer.Write(arr)
		return nil
	}

	resp.Now()
	arr, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	obj.writer.Write(arr)
	return nil
}
