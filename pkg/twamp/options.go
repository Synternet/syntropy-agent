package twamp

import "time"

type clientConfig struct {
	LocalPort     int
	PaddingSize   int
	PaddingZeroes bool
	Timeout       time.Duration
	TOS           int
}

type clientOption func(*clientConfig)

func LocalPort(port int) clientOption {
	return func(cfg *clientConfig) {
		cfg.LocalPort = port
	}
}

func Padding(padSize int) clientOption {
	return func(cfg *clientConfig) {
		cfg.PaddingSize = padSize
	}
}

func PadZeroes(zeroes bool) clientOption {
	return func(cfg *clientConfig) {
		cfg.PaddingZeroes = zeroes
	}
}

func Timeout(timeout time.Duration) clientOption {
	return func(cfg *clientConfig) {
		cfg.Timeout = timeout
	}
}

func Tos(tos int) clientOption {
	return func(cfg *clientConfig) {
		cfg.TOS = tos
	}
}
