package routestatus

const (
	statusOK    = "OK"
	statusError = "ERROR"
)

type Entry struct {
	Status  string `json:"status"`
	IP      string `json:"ip"`
	Message string `json:"msg,omitempty"`
}

func NewEntry(ip string, err error) *Entry {
	rse := &Entry{
		IP: ip,
	}

	if err == nil {
		rse.Status = statusOK
	} else {
		rse.Status = statusError
		rse.Message = err.Error()
	}
	return rse
}
