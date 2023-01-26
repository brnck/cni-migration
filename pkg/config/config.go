package config

import (
	"fmt"
	helmclient "github.com/mittwald/go-helm-client"
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

type Labels struct {
	AwsVpcCniCilium string `yaml:"aws-vpc-cni"`
	Cilium          string `yaml:"cilium"`

	Value string `yaml:"value"`
}

type Paths struct {
	KnetStress          string `yaml:"knet-stress"`
	CiliumPreMigration  string `yaml:"cilium-pre-migration"`
	CiliumPostMigration string `yaml:"cilium-post-migration"`
}

type AwsVpcCni struct {
	Namespace     string `yaml:"namespace"`
	DaemonsetName string `yaml:"daemonsetName"`
}

type ClusterAutoscaler struct {
	Namespace      string `yaml:"namespace"`
	DeploymentName string `yaml:"deploymentName"`
}

type Cilium struct {
	ReleaseName string `yaml:"release-name"`
	ChartName   string `yaml:"chart-name"`
	RepoPath    string `yaml:"repo-path"`
	Version     string `yaml:"version"`
	Namespace   string `yaml:"namespace"`
}

type Resources struct {
	DaemonSets   map[string][]string `yaml:"daemonsets"`
	Deployments  map[string][]string `yaml:"deployments"`
	StatefulSets map[string][]string `yaml:"statefulsets"`
}

type Config struct {
	*Labels            `yaml:"labels"`
	*Paths             `yaml:"paths"`
	*AwsVpcCni         `yaml:"awsVpcCni"`
	*ClusterAutoscaler `yaml:"clusterAutoscaler"`
	*Cilium            `yaml:"cilium"`
	PreflightResources *Resources `yaml:"preflightResources"`
	WatchedResources   *Resources `yaml:"watchedResources"`
	CleanUpResources   *Resources `yaml:"cleanUpResources"`

	Client     *kubernetes.Clientset
	HelmClient helmclient.Client
	Log        *logrus.Entry
}

func New(configPath string, logLevel logrus.Level, kubeFactory cmdutil.Factory) (*Config, error) {
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config path %q: %s",
			configPath, err)
	}

	config := new(Config)
	if err := yaml.UnmarshalStrict(yamlFile, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config %q: %s",
			configPath, err)
	}

	config.Client, err = kubeFactory.KubernetesClientSet()
	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes client: %s", err)
	}

	logger := logrus.New()
	logger.SetLevel(logLevel)
	config.Log = logrus.NewEntry(logger)

	hc, err := helmclient.New(&helmclient.Options{
		RepositoryConfig: "/tmp/.helmrepo",
		RepositoryCache:  "/tmp/.helmcache",
		Debug:            false,
		Linting:          false,
		Namespace:        config.Cilium.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build helm client: %s", err)
	}
	config.HelmClient = hc

	return config, nil
}
