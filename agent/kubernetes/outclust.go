package kubernetes

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"

	"golang.org/x/build/kubernetes"
)

func newOutOfCluster(namespace string) (*kubernetes.Client, error) {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "~"
	}
	/*
		configPath := homeDir + "/.kube/config"
		TODO: config parsing for file paths
	*/
	url := "https://192.168.49.2:8443"
	caCert, err := ioutil.ReadFile(homeDir + "/.minikube/ca.crt")
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(homeDir+"/.minikube/profiles/minikube/client.crt",
		homeDir+"/.minikube/profiles/minikube/client.key")
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cert},
			},
		},
	}

	return kubernetes.NewClient(url, namespace, client)
}
