package common

type Ports struct {
	TCP []uint16 `json:"tcp"`
	UDP []uint16 `json:"udp"`
}
