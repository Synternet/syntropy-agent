package docker

import (
	"context"
	"github.com/SyntropyNet/syntropy-agent/agent/common"
)

type DockerNull struct {
}

func (dn *DockerNull) ContainerInfo() []common.ServiceInfoEntry {
	return []common.ServiceInfoEntry{}
}

func (dn *DockerNull) NetworkCreate(name string, subnet string) error {
	return nil
}

func (dn *DockerNull) Name() string {
	return "DockerNull"
}

func (dn *DockerNull) Run(ctx context.Context) error {
	return nil
}
