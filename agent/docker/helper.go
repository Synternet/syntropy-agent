package docker

import (
	"context"
	"strings"

	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/docker/docker/api/types"
)

func (obj *dockerWatcher) NetworkInfo() []DockerNetworkInfoEntry {
	networkInfo := []DockerNetworkInfoEntry{}

	networks, err := obj.cli.NetworkList(context.Background(), types.NetworkListOptions{})
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

func addPort(arr *[]int, port uint16) {
	if port == 0 {
		return
	}
	for _, p := range *arr {
		if p == int(port) {
			return
		}
	}
	*arr = append(*arr, int(port))
}

func (obj *dockerWatcher) ContainerInfo() []DockerContainerInfoEntry {
	containerInfo := []DockerContainerInfoEntry{}

	containers, err := obj.cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		logger.Warning().Println(pkgName, "Container List: ", err)
		return containerInfo
	}

	for _, c := range containers {
		jsoncfg, err := obj.cli.ContainerInspect(context.Background(), c.ID)
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
			name = c.Names[0]
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
		}
		ci.Ports.TCP = []int{}
		ci.Ports.UDP = []int{}

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
