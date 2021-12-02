package docker

import (
	"errors"
	"strings"

	"github.com/docker/docker/api/types/network"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/docker/docker/api/types"
)

var errClientInit = errors.New("docker client is not initialised")

func (obj *dockerWatcher) NetworkInfo() []DockerNetworkInfoEntry {
	networkInfo := []DockerNetworkInfoEntry{}

	if obj.cli == nil {
		logger.Error().Println(pkgName, errClientInit)
		return networkInfo
	}

	networks, err := obj.cli.NetworkList(obj.ctx, types.NetworkListOptions{})
	if err != nil {
		logger.Warning().Println(pkgName, "Network List: ", err)
		return networkInfo
	}

	for _, n := range networks {
		ni := DockerNetworkInfoEntry{
			Name:    n.Name,
			ID:      n.ID,
			Subnets: []string{},
		}

		for _, netcfg := range n.IPAM.Config {
			if netcfg.Subnet != "" {
				ni.Subnets = append(ni.Subnets, netcfg.Subnet)
			}
		}

		if len(ni.Subnets) > 0 {
			networkInfo = append(networkInfo, ni)
		}
	}

	return networkInfo
}

func addPort(arr *[]uint16, port uint16) {
	if port == 0 {
		return
	}
	for _, p := range *arr {
		if p == port {
			return
		}
	}
	*arr = append(*arr, port)
}

func (obj *dockerWatcher) ContainerInfo() []DockerContainerInfoEntry {
	containerInfo := []DockerContainerInfoEntry{}

	if obj.cli == nil {
		logger.Error().Println(pkgName, errClientInit)
		return containerInfo
	}

	containers, err := obj.cli.ContainerList(obj.ctx, types.ContainerListOptions{})
	if err != nil {
		logger.Warning().Println(pkgName, "Container List: ", err)
		return containerInfo
	}

	for _, c := range containers {
		jsoncfg, err := obj.cli.ContainerInspect(obj.ctx, c.ID)
		if err != nil {
			logger.Error().Println(pkgName, "Inspect container ", c.ID, err)
		}

		var name string
		for _, env := range jsoncfg.Config.Env {
			s := strings.Split(env, "=")
			if s[0] == "SYNTROPY_SERVICE_NAME" {
				name = s[1]
				break
			}
		}
		if name == "" {
			name = jsoncfg.Config.Domainname
		}
		if name == "" {
			// why docker package returns name prepended with `/` ?
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		if name == "" {
			name = jsoncfg.Config.Hostname
		}

		ci := DockerContainerInfoEntry{
			ID:       c.ID,
			Name:     name,
			State:    c.State,
			Uptime:   c.Status,
			Networks: []string{},
			IPs:      []string{},
			Ports: common.Ports{
				TCP: []uint16{},
				UDP: []uint16{},
			},
		}

		for name, net := range c.NetworkSettings.Networks {
			if net.IPAddress != "" {
				ci.Networks = append(ci.Networks, name)
				ci.IPs = append(ci.IPs, net.IPAddress)
			}
		}

		for _, p := range c.Ports {
			switch p.Type {
			case "tcp":
				addPort(&ci.Ports.TCP, p.PrivatePort)
				addPort(&ci.Ports.TCP, p.PublicPort)
			case "udp":
				addPort(&ci.Ports.UDP, p.PrivatePort)
				addPort(&ci.Ports.UDP, p.PublicPort)
			}

		}

		if len(ci.IPs) > 0 {
			containerInfo = append(containerInfo, ci)
		}
	}

	return containerInfo
}

func (obj *dockerWatcher) NetworkCreate(name string, subnet string) error {
	if obj.cli == nil {
		return errClientInit
	}

	_, err := obj.cli.NetworkCreate(obj.ctx, name, types.NetworkCreate{
		CheckDuplicate: false,
		IPAM: &network.IPAM{
			Driver: "default",
			Config: []network.IPAMConfig{
				{
					Subnet: subnet,
				},
			},
		},
		Attachable: true,
	})
	return err
}
