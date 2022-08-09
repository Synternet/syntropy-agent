// pubip is self contained package to get host public IP
// Ip may be get from several providers (STUN and fallback to webpage currently)
// Also caches IP for some time to reduce requests to servers.
// It does not allocate separate instance for its clients so cache can be reused,
package pubip

import (
	"net"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent/pkg/pubip/stunip"
	"github.com/SyntropyNet/syntropy-agent/pkg/pubip/webip"
)

type ipProvider int

const (
	Fallback ipProvider = iota
	Stun
	WebIP
)

var publicIP struct {
	L              sync.Mutex
	ipUpdatePeriod time.Duration
	cache          struct {
		ip      net.IP
		updated time.Time
	}
	provider ipProvider
}

func init() {
	publicIP.ipUpdatePeriod = time.Minute
	publicIP.provider = Stun
}

func UpdatePeriod() time.Duration {
	return publicIP.ipUpdatePeriod
}

func SetUpdatePeriod(t time.Duration) {
	publicIP.ipUpdatePeriod = t
}

func Reset() {
	// reset IP provider and force checking
	publicIP.provider = Stun
	publicIP.cache.updated = time.Unix(0, 0)
}

func GetPublicIp() net.IP {
	publicIP.L.Lock()
	defer publicIP.L.Unlock()

	var ip net.IP
	var err error
	if time.Since(publicIP.cache.updated) > publicIP.ipUpdatePeriod {
		// Fallback means we have failed everything last time
		// Lets retry once again
		if publicIP.provider == Fallback {
			publicIP.provider = Stun
		}

		// Try STUN servers first
		if publicIP.provider == Stun {
			ip, err = stunip.PublicIP()
			if err != nil {
				// *All* STUN servers failed (internally in stunip package)
				// Reason may be - some corps have big and strict firewalls
				// Fallback to Web IP services (most probably 443 port is open)
				// And don't try stun again
				publicIP.provider = WebIP
			}
		}

		// Web service is a fallback
		if publicIP.provider == WebIP {
			ip, err = webip.PublicIP()
			if err != nil {
				// WebIP is a fallback. If it is failing - we may have some serious problems.
				// Try fallback to STUN again. But chances are low to get it working...
				publicIP.provider = Stun
			}
		}

		// Lets hope we have some result and parse them
		if err == nil {
			publicIP.cache.ip = ip
			publicIP.cache.updated = time.Now()
		} else {
			// Fallback to 0.0.0.0
			// Do not update timestamp, so I will retry asap
			// Temporary workarround until a propper solution will be implemented
			publicIP.cache.ip = net.ParseIP("0.0.0.0")
			publicIP.provider = Fallback
		}
	}

	return publicIP.cache.ip
}

func Provider() string {
	switch publicIP.provider {
	case Stun:
		return "STUN"
	case WebIP:
		return "WebIP"
	case Fallback:
		return "fallback"
	default:
		return "unknown"
	}

}
