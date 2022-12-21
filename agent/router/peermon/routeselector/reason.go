package routeselector

import "fmt"

const (
	ReasonNoChange = iota
	ReasonNewRoute
	ReasonLoss
	ReasonLatency
	ReasonRouteDelete
)

type RouteChangeReason struct {
	reason int
	oldval float32
	newval float32
}

func NewReason(r int, o, n float32) *RouteChangeReason {
	return &RouteChangeReason{
		reason: r,
		oldval: o,
		newval: n,
	}
}

func (rr *RouteChangeReason) Reason() string {
	switch rr.reason {
	case ReasonNoChange:
		return "nochange"
	case ReasonNewRoute:
		return "new"
	case ReasonLoss:
		return "loss"
	case ReasonLatency:
		return "latency"
	case ReasonRouteDelete:
		return "delete"
	default:
		return "unknown"
	}
}

func (rr *RouteChangeReason) Value() float32 {
	// TODO: should I return new value, diff or both ?
	return rr.newval
}

func (rr *RouteChangeReason) String() string {
	return fmt.Sprintf("%s: %f vs %f", rr.Reason(), rr.oldval, rr.newval)
}
