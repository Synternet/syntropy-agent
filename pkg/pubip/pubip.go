// pubip is self contained package to get host public IP
// Ip may be get from several providers (STUN and fallback to webpage currently)
// Also caches IP for some time to reduce requests to servers.
// It does not allocate separate instance for its clients so cache can be reused,
package pubip

import (
	"net"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/pubip/stunip"
	"github.com/SyntropyNet/syntropy-agent/pkg/pubip/webip"
)

const (
	providerStun = iota
	providerWeb
)
const pkgName = "PublicIP. "

var publicIP struct {
	L              sync.Mutex
	ipUpdatePeriod time.Duration
	cache          struct {
		ip      net.IP
		updated time.Time
	}
	provider int
}

func init() {
	publicIP.ipUpdatePeriod = 10 * time.Minute
	publicIP.provider = providerStun
}

func UpdatePeriod() time.Duration {
	return publicIP.ipUpdatePeriod
}

func SetUpdatePeriod(t time.Duration) {
	publicIP.ipUpdatePeriod = t
}

func GetPublicIp() net.IP {
	publicIP.L.Lock()
	defer publicIP.L.Unlock()

	var ip net.IP
	var err error
	if time.Since(publicIP.cache.updated) > publicIP.ipUpdatePeriod {
		// Try STUN servers first
		if publicIP.provider == providerStun {
			ip, err = stunip.PublicIP()
			if err == nil {
				logger.Info().Println(pkgName, "Public IP (STUN):", ip.String())
			} else {
				// *All* STUN servers failed (internally in stunip package)
				// Reason may be - some corps have big and strict firewalls
				// Fallback to Web IP services (most probably 443 port is open)
				// And don't try stun again
				publicIP.provider = providerWeb
				// TODO think about configurable logger here and not use agent's logger
				// This would increase package reusability
				logger.Warning().Println(pkgName, "STUN failed", err, ". Fallback to WebIP getting.")
			}
		}

		// Web service is a fallback
		if publicIP.provider == providerWeb {
			ip, err = webip.PublicIP()
			if err == nil {
				logger.Info().Println(pkgName, "Public IP (Web):", ip.String())
			} else {
				// WebIP is a fallback. If it is failing - we may have some serious problems.
				// Try fallback to STUN again. But chances are low to get it working...
				publicIP.provider = providerStun
				// TODO think about configurable logger here and not use agent's logger
				// This would increase package reusability
				logger.Error().Println(pkgName, "WebIP failed:", err, ". Will (re)try STUN next time.")
			}
		}

		// Lets hope we have some result and parse them
		if err == nil {
			publicIP.cache.ip = ip
			publicIP.cache.updated = time.Now()
		} else {
			if publicIP.cache.ip == nil {
				// Fallback to 0.0.0.0, if have no valid older ip (stick to old value, if it is present)
				// Do not update timestamp, so I will retry asap
				// Temporary workarround until a propper solution will be implemented
				publicIP.cache.ip = net.ParseIP("0.0.0.0")
			}
		}
	}

	return publicIP.cache.ip
}
