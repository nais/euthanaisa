package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/nais/euthanaisa/internal/client"
	"github.com/nais/euthanaisa/internal/config"
	"github.com/nais/euthanaisa/internal/euthanaiser"
	"github.com/nais/euthanaisa/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	exitCodeSuccess = iota
	exitCodeRunError
	exitCodeLoggerError
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := logrus.StandardLogger()

	err := godotenv.Load()
	if err != nil {
		l.WithError(err).Warnf("error when loading .env file, continuing with default environment variables")
	}

	cfg, err := config.NewConfig()
	if err != nil {
		l.WithError(err).Errorf("error when processing configuration")
		os.Exit(exitCodeRunError)
	}

	appLog := setupLogger(l, cfg.LogFormat, cfg.LogLevel)

	appLog.Infof("Starting euthanaisa with log level %s", cfg.LogLevel)

	kubeConfig, err := kubeconfig()
	if err != nil {
		appLog.WithError(err).Errorf("error when getting kubeconfig")
		os.Exit(exitCodeRunError)
	}

	registry := prometheus.NewRegistry()

	clients, err := client.LoadResourceClients(cfg.Resources, kubeConfig, registry, appLog.WithField("sub-system", "resource-client"))
	if err != nil {
		appLog.WithError(err).Errorf("error when loading resource clients")
		os.Exit(exitCodeRunError)
	}

	e, err := euthanaiser.New(cfg, clients, registry, appLog.WithField("system", "euthanaisa"))
	if err != nil {
		appLog.WithError(err).Errorf("error when initializing euthanaisa")
		os.Exit(exitCodeRunError)
	}

	e.Run(ctx)
	os.Exit(exitCodeSuccess)
}

func setupLogger(log *logrus.Logger, logFormat, logLevel string) logrus.FieldLogger {
	appLogger, err := logger.New(logFormat, logLevel)
	if err != nil {
		log.WithError(err).Errorf("error when creating application logger")
		os.Exit(exitCodeLoggerError)
	}

	return appLogger
}

func kubeconfig() (*rest.Config, error) {
	if kConfig := os.Getenv("KUBECONFIG"); kConfig != "" {
		return clientcmd.BuildConfigFromFlags("", kConfig)
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
