package routestatus

import "net/netip"

const (
	statusOK    = "OK"
	statusError = "ERROR"
)

type Entry struct {
	Status  string `json:"status"`
	IP      string `json:"ip"`
	Message string `json:"msg,omitempty"`
}

func NewEntry(ip netip.Prefix, err error) *Entry {
	rse := &Entry{
		IP: ip.String(),
	}

	if err == nil {
		rse.Status = statusOK
	} else {
		rse.Status = statusError
		rse.Message = err.Error()
	}
	return rse
}
