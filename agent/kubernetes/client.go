package kubernetes

import (
	"net"
	"os"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	portTCP = "TCP"
	portUDP = "UDP"
)

func (obj *kubernet) initClient() bool {
	logger.Info().Println(pkgName, "trying in cluster config")
	kConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Error().Println(pkgName, "in cluster config", err)
		logger.Info().Println(pkgName, "try fallback to out of cluster config")
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "~"
		}
		configPath := homeDir + "/.kube/config"
		kConfig, err = clientcmd.BuildConfigFromFlags("", configPath)
		if err != nil {
			logger.Error().Println(pkgName, "out of cluster config", err)
			return false
		}
	}

	obj.klient, err = kubernetes.NewForConfig(kConfig)
	if err != nil {
		logger.Error().Println(pkgName, "kubernetes client", err)
		return false
	}
	return true
}

// Be sure to call initClient() before
// Caller is responsible to be sure that obj.klient is not nil
func (obj *kubernet) monitorServices() []kubernetesServiceEntry {
	res := []kubernetesServiceEntry{}
	srvs, err := obj.klient.CoreV1().Services(config.GetNamespace()).List(obj.ctx, metav1.ListOptions{})

	if err != nil {
		logger.Error().Println(pkgName, "listing services", err)
		return res
	}

	for _, srv := range srvs.Items {
		ip := net.ParseIP(srv.Spec.ClusterIP)
		if ip == nil {
			// kubernetes documentation says that ClusterIP may be IP string,
			// empty string "" or "None". Ignore non valid IPs.
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
