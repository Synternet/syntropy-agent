package kubernetes

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"golang.org/x/build/kubernetes"
	"gopkg.in/yaml.v3"
)

type clientConfig struct {
	BaseURL        string
	CaAuthFile     string
	ClientCertFile string
	ClientKeyFile  string
}

func newOutOfCluster(namespace string) (*kubernetes.Client, error) {
	config, err := parseConfig()
	if err != nil {
		return nil, fmt.Errorf("kubernetes config parsing: %s", err)
	}

	caCert, err := ioutil.ReadFile(config.CaAuthFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(config.ClientCertFile, config.ClientKeyFile)
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

	return kubernetes.NewClient(config.BaseURL, namespace, client)
}

type clusterEntry struct {
	CAAuth string `yaml:"certificate-authority"`
	Server string `yaml:"server"`
}

type userEntry struct {
	Certificate string `yaml:"client-certificate"`
	Key         string `yaml:"client-key"`
}

type kubeConfig struct {
	ApiVersion string `yaml:"apiVersion"`
	Clusters   []struct {
		Name    string       `yaml:"name"`
		Cluster clusterEntry `yaml:"cluster"`
	} `yaml:"clusters"`
	Users []struct {
		Name string    `yaml:"name"`
		User userEntry `yaml:"user"`
	} `yaml:"users"`
}

func parseConfig() (*clientConfig, error) {
	configFileName := os.Getenv("KUBECONFIG")
	if len(configFileName) == 0 {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "~"
		}
		configFileName = homeDir + "/.kube/config"
	}

	configFile, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return nil, err
	}

	var kcfg kubeConfig
	err = yaml.Unmarshal(configFile, &kcfg)
	if err != nil {
		return nil, err
	}

	if len(kcfg.Clusters) == 0 {
		return nil, fmt.Errorf("missing clusters configuration")
	}
	if len(kcfg.Users) == 0 {
		return nil, fmt.Errorf("missing users configuration")
	}

	return &clientConfig{
		BaseURL:        kcfg.Clusters[0].Cluster.Server,
		CaAuthFile:     kcfg.Clusters[0].Cluster.CAAuth,
		ClientCertFile: kcfg.Users[0].User.Certificate,
		ClientKeyFile:  kcfg.Users[0].User.Key,
	}, nil
}
