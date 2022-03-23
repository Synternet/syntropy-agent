package twamp

type clientConfig struct {
	LocalPort int
	Padding   int
	Timeout   int
	TOS       int
}

type clientOption func(*clientConfig)

func LocalPort(port int) clientOption {
	return func(cfg *clientConfig) {
		cfg.LocalPort = port
	}
}

func Padding(padSize int) clientOption {
	return func(cfg *clientConfig) {
		cfg.Padding = padSize
	}
}

func Timeout(timeout int) clientOption {
	return func(cfg *clientConfig) {
		cfg.Timeout = timeout
	}
}

func Tos(tos int) clientOption {
	return func(cfg *clientConfig) {
		cfg.TOS = tos
	}
}
