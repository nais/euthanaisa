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
	"github.com/nais/euthanaisa/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
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
		l.WithError(err).Warnf(".env file not found, continuing with default envs")
	}

	cfg, err := config.NewConfig()
	if err != nil {
		l.WithError(err).Errorf("error when processing configuration")
		os.Exit(exitCodeRunError)
	}

	appLog := setupLogger(l, cfg.LogFormat, cfg.LogLevel)

	appLog.WithField("system", "main").Infof("Starting euthanaisa with log level %s and format %s", cfg.LogLevel, cfg.LogFormat)

	kubeConfig, err := kubeconfig()
	if err != nil {
		appLog.WithError(err).Errorf("error when getting kubeconfig")
		os.Exit(exitCodeRunError)
	}

	dynClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		appLog.WithError(err).Errorf("error when creating dynamic client")
		os.Exit(exitCodeRunError)
	}

	registry := prometheus.NewRegistry()
	pusher := metrics.Register(cfg.PushgatewayURL, registry)

	factory := client.NewFactory(dynClient, appLog.WithField("system", "client-factory"))
	ownerClients, handlerByKind, err := factory.BuildClients(cfg.Resources)
	if err != nil {
		appLog.WithError(err).Errorf("error when building resource clients")
		os.Exit(exitCodeRunError)
	}

	e := euthanaiser.New(ownerClients, handlerByKind, pusher, appLog.WithField("system", "euthanaisa"))
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
