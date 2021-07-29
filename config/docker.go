package config

import (
	"context"
	"log"
	"os"
	"strings"

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

func initContainer() {
	log.Println("Init Docker container")
	cache.containerType = strings.ToLower(os.Getenv("SYNTROPY_NETWORK_API"))
	initDockerNetInfo()
	initDockerContainerInfo()
	// TODO: shedule docker changes subscribe and monitor
}

func isDockerContainer() bool {
	return cache.containerType == "docker"
}

func initDockerNetInfo() {
	cache.docker.networkInfo = []DockerNetworkInfoEntry{}
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
			Name: n.Name,
			ID:   n.ID,
		}
		for _, netcfg := range n.IPAM.Config {
			if netcfg.Subnet != "" {
				ni.Subnets = append(ni.Subnets, netcfg.Subnet)
			}

		}
		if len(ni.Subnets) > 0 {
			cache.docker.networkInfo = append(cache.docker.networkInfo, ni)
		}
	}

}

func initDockerContainerInfo() {
	cache.docker.containerInfo = []DockerContainerInfoEntry{}
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

		cache.docker.containerInfo = append(cache.docker.containerInfo, ci)
	}
}
