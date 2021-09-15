package docker

import "github.com/SyntropyNet/syntropy-agent-go/pkg/common"

type DockerHelper interface {
	NetworkInfo() []DockerNetworkInfoEntry
	ContainerInfo() []DockerContainerInfoEntry
}

type DockerService interface {
	common.Service
	DockerHelper
}
