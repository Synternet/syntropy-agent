package common

type Router interface {
	RouteAdd(ifname string, gw string, ips ...string) error
	RouteDel(ifname string, ips ...string) error
}
