package kubernetes

import (
	"context"
	"os"

	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
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
	srvs, err := obj.klient.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logger.Error().Println(pkgName, "listing services", err)
	}
	for _, srv := range srvs.Items {
		e := kubernetesServiceEntry{
			Name:    srv.Name,
			Subnets: make([]string, len(srv.Spec.ClusterIPs)),
			Ports: common.Ports{
				TCP: []uint16{},
				UDP: []uint16{},
			},
		}
		copy(e.Subnets, srv.Spec.ClusterIPs)

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
