package routeselector

type RouteSelectorConfig struct {
	AverageSize              uint
	RouteStrategy            int
	RerouteRatio             float32
	RerouteDiff              float32
	RouteDeleteLossThreshold float32
}
