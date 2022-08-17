package kubernetes

import (
	"fmt"
	"net"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"golang.org/x/build/kubernetes"
)

const (
	portTCP = "TCP"
	portUDP = "UDP"
)

func (obj *kubernet) initClient() error {
	obj.namespaces = config.GetNamespace()
	if len(obj.namespaces) == 0 {
		return fmt.Errorf("SYNTROPY_NAMESPACE is not set")
	}

	inErr := obj.initInCluster()
	if inErr == nil {
		logger.Info().Println(pkgName, "using in cluster config")
		return nil
	}

	outErr := obj.initOutOfCluster()
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
func (obj *kubernet) monitorServices() ([]kubernetesServiceEntry, error) {
	res := []kubernetesServiceEntry{}

	for _, namespace := range obj.namespaces {
		klient, err := kubernetes.NewClient(obj.baseURL, namespace, obj.httpClient)
		if err != nil {
			// failed initialise client for one namespace
			// note error and try other namespaces
			logger.Error().Println(pkgName, namespace, "kubernetes client error:", err)
			// kubernetes client create failed - do nothing with it
			continue
		}

		srvs, err := klient.GetServices(obj.ctx)
		if err != nil {
			// failed getting services for one namespace
			// note error and try other namespaces
			logger.Error().Println(pkgName, namespace, "Get Services error:", err)
			// close kubernetes client and continue on next namespace
			klient.Close()
			continue
		}

		// parse and add the services
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

		// one namespace is done, close kubernetes client
		klient.Close()
	}

	return res, nil
}
