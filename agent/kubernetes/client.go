package kubernetes

import (
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
	}
	for _, srv := range srvs.Items {
		if len(srv.Spec.ClusterIPs) == 0 {
			continue
		}
		e := kubernetesServiceEntry{
			Name:   srv.Name,
			Subnet: srv.Spec.ClusterIPs[0],
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
