package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Resource interface {
	GetGroup() string
	GetVersion() string
	GetResourceName() string
}

type resourceConfig struct {
	Group    string `yaml:"group"`
	Version  string `yaml:"version"`
	Resource string `yaml:"resource"`
}

func (r *resourceConfig) GetGroup() string        { return r.Group }
func (r *resourceConfig) GetVersion() string      { return r.Version }
func (r *resourceConfig) GetResourceName() string { return r.Resource }

type Config struct {
	Resources      []Resource
	PushgatewayURL string `envconfig:"PUSHGATEWAY_URL"`
	LogLevel       string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat      string `envconfig:"LOG_FORMAT" default:"json"`
	ResourcesFile  string `envconfig:"RESOURCES_FILE" default:"/app/config/resources.yaml"`
}

func NewConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env file found")
	}

	cfg := &Config{}
	err = envconfig.Process("", cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	cfg.loadResources()

	return cfg, nil
}

func (c *Config) loadResources() {
	b, err := os.ReadFile(c.ResourcesFile)
	if err != nil {
		log.Fatalf("failed to read resources file %s: %v", c.ResourcesFile, err)
	}
	var configs []resourceConfig
	if err := yaml.Unmarshal(b, &configs); err != nil {
		log.Fatalf("failed to unmarshal resources file %s: %v", c.ResourcesFile, err)
	}

	c.Resources = make([]Resource, len(configs))
	for i, rc := range configs {
		c.Resources[i] = &rc
	}
}

func Kubeconfig() (*rest.Config, error) {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		kubeconfigPath := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(kubeconfigPath); err == nil {
			return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		}
	}

	return rest.InClusterConfig()
}
