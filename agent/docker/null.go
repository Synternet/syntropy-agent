package docker

import "context"

type DockerNull struct {
}

func (dn *DockerNull) NetworkInfo() []DockerNetworkInfoEntry {
	return []DockerNetworkInfoEntry{}
}

func (dn *DockerNull) ContainerInfo() []DockerContainerInfoEntry {
	return []DockerContainerInfoEntry{}
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
