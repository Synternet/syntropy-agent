// pubip is self contained package to get host public IP
// Ip may be get from several providers (STUN and fallback to webpage currently)
// Also caches IP for some time to reduce requests to servers.
// It does not allocate separate instance for its clients so cache can be reused,
package pubip

import (
	"net"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/pubip/stunip"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/pubip/webip"
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
			if err != nil {
				// *All* STUN servers failed (internally in stunip package)
				// Reason may be - some corps have big and strict firewalls
				// Fallback to Web IP services (most probably 443 port is open)
				// And don't try stun again
				publicIP.provider = providerWeb
				// TODO think about configurable logger here and not use agent's logger
				// This would increase package reusability
				logger.Warning().Println(pkgName, "STUN failed. Fallback to WebIP getting.")
			}
		}

		// Web service is a fallback
		if publicIP.provider == providerWeb {
			ip, err = webip.PublicIP()
		}

		// Lets hope we have some result and parse them
		if err == nil {
			publicIP.cache.ip = ip
			publicIP.cache.updated = time.Now()
		} else {
			logger.Error().Println(pkgName, err)
		}
	}

	return publicIP.cache.ip
}
