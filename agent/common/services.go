package common

type ServiceInfoMessage struct {
	MessageHeader
	Data []ServiceInfoEntry `json:"data"`
}

type ServiceInfoEntry struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	IPs      []string `json:"ips"`
	Networks []string `json:"network"`
	Ports    Ports    `json:"ports"`
}
