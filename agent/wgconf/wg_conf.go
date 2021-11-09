package wgconf

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/agent/common"
	"github.com/SyntropyNet/syntropy-agent-go/agent/peeradata"
	"github.com/SyntropyNet/syntropy-agent-go/agent/router"
	"github.com/SyntropyNet/syntropy-agent-go/agent/routestatus"
	"github.com/SyntropyNet/syntropy-agent-go/agent/swireguard"
	"github.com/SyntropyNet/syntropy-agent-go/internal/env"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
)

const (
	pkgName = "Wg_Conf. "
	cmd     = "WG_CONF"
)

type wgConf struct {
	writer io.Writer
	wg     *swireguard.Wireguard
	router *router.Router
}

// This struct is not used in Linux agent
// Is is intendended only for desktop (MacOS and Windows) application
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
		GroupID          int                   `json:"connection_group_id,omitempty"`
		AgentID          int                   `json:"agent_id,omitempty"`
		AllowedIPsInfo   []allowedIPsInfoEntry `json:"allowed_ips_info,omitempty"`
	}
}

func (e *wgConfEntry) asPeerInfo() *swireguard.PeerInfo {
	var ifname string
	if strings.HasPrefix(e.Args.IfName, env.InterfaceNamePrefix) {
		ifname = e.Args.IfName
	} else {
		ifname = env.InterfaceNamePrefix + e.Args.IfName
	}

	return &swireguard.PeerInfo{
		IfName:       ifname,
		IP:           e.Args.EndpointIPv4,
		PublicKey:    e.Args.PublicKey,
		ConnectionID: e.Metadata.ConnectionID,
		GroupID:      e.Metadata.GroupID,
		AgentID:      e.Metadata.AgentID,
		Port:         e.Args.EndpointPort,
		Gateway:      e.Args.GatewayIPv4,
		AllowedIPs:   e.Args.AllowedIPs,
	}
}

func (e *wgConfEntry) asInterfaceInfo() *swireguard.InterfaceInfo {
	var ifname string
	if strings.HasPrefix(e.Args.IfName, env.InterfaceNamePrefix) {
		ifname = e.Args.IfName
	} else {
		ifname = env.InterfaceNamePrefix + e.Args.IfName
	}

	return &swireguard.InterfaceInfo{
		IfName:    ifname,
		IP:        e.Args.IP,
		PublicKey: e.Args.PublicKey,
		Port:      e.Args.Port,
	}
}

type wgConfMsg struct {
	common.MessageHeader
	Data []wgConfEntry `json:"data"`
}

func (msg *wgConfMsg) AddPeerCmd(cmd string, pi *swireguard.PeerInfo) {
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

func (msg *wgConfMsg) AddInterfaceCmd(cmd string, ii *swireguard.InterfaceInfo) {
	e := wgConfEntry{
		Function: cmd,
	}
	e.Args.IfName = ii.IfName
	e.Args.IP = ii.IP
	e.Args.PublicKey = ii.PublicKey
	e.Args.Port = ii.Port

	msg.Data = append(msg.Data, e)
}

func New(w io.Writer, wg *swireguard.Wireguard, r *router.Router) common.Command {
	return &wgConf{
		writer: w,
		wg:     wg,
		router: r,
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

	routeStatus := routestatus.NewMsg()
	padMsg := peeradata.NewMessage()

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
			if err == nil {
				routeRes, peersData := obj.router.RouteAdd(
					&common.SdnNetworkPath{
						Ifname:       cmd.Args.IfName,
						Gateway:      cmd.Args.GatewayIPv4,
						ConnectionID: cmd.Metadata.ConnectionID,
						GroupID:      cmd.Metadata.GroupID,
					}, cmd.Args.AllowedIPs)
				routeStatus.Add(cmd.Metadata.ConnectionID, cmd.Metadata.GroupID, routeRes)
				padMsg.Add(peersData...)
			}

		case "remove_peer":
			// Nobody is interested in RouteDel results
			obj.router.RouteDel(
				&common.SdnNetworkPath{
					Ifname: cmd.Args.IfName,
				}, cmd.Args.AllowedIPs)

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
		errResp := common.ErrorResponce{
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

	resp.Now()
	arr, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	logger.Debug().Println(pkgName, "Sending: ", string(raw))
	obj.writer.Write(arr)

	routeStatus.Send(obj.writer)
	padMsg.Send(obj.writer)

	return nil
}
