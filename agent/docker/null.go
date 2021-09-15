package docker

type DockerNull struct {
}

func (dn *DockerNull) NetworkInfo() []DockerNetworkInfoEntry {
	return []DockerNetworkInfoEntry{}
}

func (dn *DockerNull) ContainerInfo() []DockerContainerInfoEntry {
	return []DockerContainerInfoEntry{}
}

func (dn *DockerNull) Name() string {
	return "DockerNull"
}

func (dn *DockerNull) Start() error {
	return nil
}

func (dn *DockerNull) Stop() error {
	return nil
}
