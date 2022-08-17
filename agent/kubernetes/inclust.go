package kubernetes

import (
	"fmt"
	"net"
	"net/http"
	"os"
)

func (obj *kubernet) initInCluster() error {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return fmt.Errorf("not in cluster (missing env variables)")
	}

	transport, err := newTokenRoundTripper()
	if err != nil {
		return fmt.Errorf("http transport: %s", err)
	}

	obj.baseURL = "https://" + net.JoinHostPort(host, port)
	obj.httpClient = &http.Client{
		Transport: transport,
	}

	return nil
}
