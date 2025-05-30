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

type Config struct {
	Resources      []ResourceConfig
	PushgatewayURL string `envconfig:"PUSHGATEWAY_URL"`
	LogLevel       string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat      string `envconfig:"LOG_FORMAT" default:"json"`
	ResourcesFile  string `envconfig:"RESOURCES_FILE" default:"/app/config/resources.yaml"`
}

type ResourceConfig struct {
	Group    string `yaml:"group"`
	Version  string `yaml:"version"`
	Resource string `yaml:"resource"`
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
	var configs []ResourceConfig
	if err := yaml.Unmarshal(b, &configs); err != nil {
		log.Fatalf("failed to unmarshal resources file %s: %v", c.ResourcesFile, err)
	}
	c.Resources = configs
}

func Kubeconfig() (*rest.Config, error) {
	// Use KUBECONFIG if explicitly set
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	// Fallback to default kubeconfig location (~/.kube/config)
	home, err := os.UserHomeDir()
	if err == nil {
		kubeconfigPath := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(kubeconfigPath); err == nil {
			return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		}
	}

	// Fallback to in-cluster config
	return rest.InClusterConfig()
}
