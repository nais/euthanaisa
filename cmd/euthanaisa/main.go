package main

import (
	"context"
	"os"

	"github.com/nais/euthanaisa/internal/logger"

	"github.com/joho/godotenv"
	"github.com/nais/euthanaisa/internal/config"
	"github.com/nais/euthanaisa/internal/euthanaiser"
	"github.com/sirupsen/logrus"
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

	e, err := euthanaiser.New(cfg, appLog.WithField("application", "euthanaisa"))
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
