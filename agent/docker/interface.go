package docker

import "github.com/SyntropyNet/syntropy-agent/agent/common"

type DockerHelper interface {
	NetworkInfo() []DockerNetworkInfoEntry
	ContainerInfo() []DockerContainerInfoEntry
	NetworkCreate(name string, subnet string) error
}

type DockerService interface {
	common.Service
	DockerHelper
}
