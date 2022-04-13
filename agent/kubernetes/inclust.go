package kubernetes

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"golang.org/x/build/kubernetes"
)

func newInCluster(namespace string) (*kubernetes.Client, error) {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return nil, fmt.Errorf("not in cluster (missing env variables)")
	}
	baseURL := "https://" + net.JoinHostPort(host, port)

	transport, err := newTokenRoundTripper()
	if err != nil {
		return nil, fmt.Errorf("http transport: %s", err)
	}

	client := &http.Client{
		Transport: transport,
	}

	return kubernetes.NewClient(baseURL, namespace, client)
}
