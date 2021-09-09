package docker

type DockerNull struct {
}

func (dn *DockerNull) NetworkInfo() []DockerNetworkInfoEntry {
	return []DockerNetworkInfoEntry{}
}

func (dn *DockerNull) ContainerInfo() []DockerContainerInfoEntry {
	return []DockerContainerInfoEntry{}
}
