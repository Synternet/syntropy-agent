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

func initContainer() {
	log.Println("Init Docker container")
	cache.containerType = strings.ToLower(os.Getenv("SYNTROPY_NETWORK_API"))
	initDockerNetInfo()
}

func isDockerContainer() bool {
	return cache.containerType == "docker"
}

func initDockerNetInfo() {
	cache.dockerNetInfo = []DockerNetworkInfoEntry{}
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
			cache.dockerNetInfo = append(cache.dockerNetInfo, ni)
		}
	}

}

func getDockerContainerInfo() {
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
	}

	for _, container := range containers {
		log.Printf("Container info %+v\n\t\nNetInfo: %+v\n\n", container, container.NetworkSettings.Networks)
	}
}
