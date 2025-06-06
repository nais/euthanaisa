package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Resources      []ResourceConfig
	PushgatewayURL string `envconfig:"PUSHGATEWAY_URL"`
	LogLevel       string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat      string `envconfig:"LOG_FORMAT" default:"json"`
	ResourcesFile  string `envconfig:"RESOURCES_FILE" default:"/app/config/resources.yaml"`
}

type ResourceConfig struct {
	Group    string   `yaml:"group"`
	Version  string   `yaml:"version"`
	Kind     string   `yaml:"kind"`
	Resource string   `yaml:"resource"`
	OwnedBy  []string `yaml:"ownedBy"`
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

	if err = cfg.loadResources(); err != nil {
		return nil, fmt.Errorf("failed to load resources: %w", err)
	}

	return cfg, nil
}

func (c *Config) loadResources() error {
	b, err := os.ReadFile(c.ResourcesFile)
	if err != nil {
		return fmt.Errorf("failed to read resources file %s: %w", c.ResourcesFile, err)
	}
	var configs []ResourceConfig
	if err = yaml.Unmarshal(b, &configs); err != nil {
		return fmt.Errorf("failed to unmarshal resources from file %s: %w", c.ResourcesFile, err)
	}

	c.Resources = make([]ResourceConfig, len(configs))
	copy(c.Resources, configs)
	return nil
}
