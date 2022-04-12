package kubernetes

import (
	"fmt"
	"net"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

const (
	portTCP = "TCP"
	portUDP = "UDP"
)

func (obj *kubernet) initClient() error {
	namespace := config.GetNamespace()
	if len(namespace) == 0 {
		return fmt.Errorf("SYNTROPY_NAMESPACE is not set")
	}

	var inErr, outErr error
	obj.klient, inErr = newInCluster(namespace)

	if inErr == nil {
		logger.Info().Println(pkgName, "using in cluster config")
		return nil
	}

	obj.klient, outErr = newOutOfCluster(namespace)
	if outErr == nil {
		logger.Info().Println(pkgName, "using out of cluster config")
		return nil
	}

	logger.Error().Println(pkgName, "in cluster:", inErr)
	logger.Error().Println(pkgName, "out of cluster:", inErr)

	return fmt.Errorf("could not initialise kubernetes client")
}

// Be sure to call initClient() before
// Caller is responsible to be sure that obj.klient is not nil
func (obj *kubernet) monitorServices() []kubernetesServiceEntry {
	res := []kubernetesServiceEntry{}
	srvs, err := obj.klient.GetServices(obj.ctx)

	if err != nil {
		logger.Error().Println(pkgName, "listing services", err)
		return res
	}

	for _, srv := range srvs {
		ip := net.ParseIP(srv.Spec.ClusterIP)
		if ip == nil {
			// Ignore non valid IPs (may be empty string "" or "none")
			continue
		}

		e := kubernetesServiceEntry{
			Name:   srv.Name,
			Subnet: srv.Spec.ClusterIP,
			Ports: common.Ports{
				TCP: []uint16{},
				UDP: []uint16{},
			},
		}

		for _, port := range srv.Spec.Ports {
			switch port.Protocol {
			case portTCP:
				e.Ports.TCP = append(e.Ports.TCP, uint16(port.Port))
			case portUDP:
				e.Ports.UDP = append(e.Ports.UDP, uint16(port.Port))
			}
		}
		res = append(res, e)
	}

	return res
}
