package pubip

import (
	"net"
	"time"
)

var ipUpdatePeriod = 10 * time.Minute
var cache struct {
	ip      net.IP
	updated time.Time
}

func UpdatePeriod() time.Duration {
	return ipUpdatePeriod
}

func SetUpdatePeriod(t time.Duration) {
	ipUpdatePeriod = t
}

func GetPublicIp() net.IP {
	if time.Since(cache.updated) > ipUpdatePeriod {
		ip, err := myPublicIp()
		// TODO think about configurable logger here
		if err == nil {
			cache.ip = ip
			cache.updated = time.Now()
		}
	}

	return cache.ip
}
