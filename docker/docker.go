package docker

import (
	"context"
	"log"

	"github.com/SyntropyNet/syntropy-agent-go/config"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type DockerNetworkInfoEntry struct {
	ID      string   `json:"agent_network_id"`
	Name    string   `json:"agent_network_name"`
	Subnets []string `json:"agent_network_subnets"`
}

type DockerContainerInfoEntry struct {
	ID       string   `json:"agent_container_id"`
	Name     string   `json:"agent_container_name"`
	State    string   `json:"agent_container_state"`
	Uptime   string   `json:"agent_container_uptime"`
	Networks []string `json:"agent_container_networks"`
	IPs      []string `json:"agent_container_ips"`
	Ports    struct {
		TCP []int `json:"tcp"`
		UDP []int `json:"udp"`
	} `json:"agent_container_ports"`
}

func isDockerContainer() bool {
	return config.GetContainerType() == "docker"
}

func NetworkInfo() (networkInfo []DockerNetworkInfoEntry) {
	networkInfo = []DockerNetworkInfoEntry{}
	if !isDockerContainer() {
		return
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Println(err)
		return
	}
	defer cli.Close()

	networks, err := cli.NetworkList(context.Background(), types.NetworkListOptions{})
	if err != nil {
		log.Println(err)
		return
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

func ContainerInfo() (containerInfo []DockerContainerInfoEntry) {
	containerInfo = []DockerContainerInfoEntry{}
	if !isDockerContainer() {
		return
	}

	log.Println("GetDocker Container Info")
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Println(err)
		return
	}
	defer cli.Close()

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		log.Println(err)
		return
	}

	addPort := func(arr *[]int, port uint16) {
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

	for _, c := range containers {
		//name := c.HostConfig.

		ci := DockerContainerInfoEntry{
			ID:       c.ID,
			Name:     c.Names[0], // TODO: is this correct ?
			State:    c.State,
			Uptime:   c.Status,
			Networks: []string{},
			IPs:      []string{},
		}
		ci.Ports.TCP = []int{}
		ci.Ports.UDP = []int{}

		// TODO: Add network names, IPs and ports info
		log.Printf("Container info %+v\n\t\nNetInfo: %+v\n\n", c, c.NetworkSettings.Networks)

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

		containerInfo = append(containerInfo, ci)
	}

	return containerInfo
}
