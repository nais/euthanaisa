package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/nais/euthanaisa/internal/client"
	"github.com/nais/euthanaisa/internal/euthanaiser"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type resource struct {
	Group    string `yaml:"group"`
	Version  string `yaml:"version"`
	Resource string `yaml:"resource"`
}

func main() {
	ctx := context.Background()

	logLevel := getEnv("LOG_LEVEL", "info")
	logFormat := getEnv("LOG_FORMAT", "json")
	resourcesFile := getEnv("RESOURCES_FILE", "/app/config/resources.yaml")
	pushgatewayEndpoint := os.Getenv("PUSHGATEWAY_ENDPOINT")

	setupLogging(logLevel, logFormat)
	slog.Info("starting euthanaisa", "logLevel", logLevel)

	resources, err := loadResources(resourcesFile)
	if err != nil {
		slog.Error("loading resources", "error", err)
		os.Exit(1)
	}

	kubeConfig, err := kubeconfig()
	if err != nil {
		slog.Error("getting kubeconfig", "error", err)
		os.Exit(1)
	}

	dynClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		slog.Error("creating dynamic client", "error", err)
		os.Exit(1)
	}

	clients := client.BuildClients(dynClient, resources)

	e := euthanaiser.New(clients)
	e.Run(ctx)

	pushMetrics(ctx, pushgatewayEndpoint)
}

func loadResources(path string) ([]client.Resource, error) {
	b, err := os.ReadFile(path) // #nosec G304 -- path from trusted env var
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var resources []resource
	if err = yaml.Unmarshal(b, &resources); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	result := make([]client.Resource, len(resources))
	for i, r := range resources {
		result[i] = client.Resource{Group: r.Group, Version: r.Version, Resource: r.Resource}
	}
	return result, nil
}

func pushMetrics(ctx context.Context, endpoint string) {
	if endpoint == "" {
		return
	}
	slog.Info("pushing metrics", "endpoint", endpoint)
	if err := push.New(endpoint, "euthanaisa").Gatherer(prometheus.DefaultGatherer).AddContext(ctx); err != nil {
		slog.Error("pushing metrics", "error", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func setupLogging(level, format string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	slog.SetDefault(slog.New(handler))
}

func kubeconfig() (*rest.Config, error) {
	if kConfig := os.Getenv("KUBECONFIG"); kConfig != "" {
		return clientcmd.BuildConfigFromFlags("", kConfig)
	}

	if home, err := os.UserHomeDir(); err == nil {
		path := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(path); err == nil {
			return clientcmd.BuildConfigFromFlags("", path)
		}
	}

	return rest.InClusterConfig()
}
