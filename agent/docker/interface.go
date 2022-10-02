package docker

import "github.com/SyntropyNet/syntropy-agent/agent/common"

type DockerHelper interface {
	ContainerInfo() []common.ServiceInfoEntry
	NetworkCreate(name string, subnet string) error
}

type DockerService interface {
	common.Service
	DockerHelper
}
