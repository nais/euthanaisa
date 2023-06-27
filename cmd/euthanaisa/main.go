package main

import (
	"context"
	"flag"
	"os"

	"github.com/nais/euthanaisa/internal/euthanaiser"
	log "github.com/sirupsen/logrus"
)

var (
	logLevel       string
	pushgatewayURL string
)

func init() {
	flag.StringVar(&logLevel, "log-level", envOrDefault("LOG_LEVEL", "info"), "Application log level")
	flag.StringVar(&pushgatewayURL, "pushgateway-url", envOrDefault("PUSHGATEWAY_URL", "http://prometheus-pushgateway.nais-system:9091"), "Pushgateway URL")
}

func main() {
	flag.Parse()
	log := setupLogger(logLevel)
	log.Infof("Starting euthanaisa with log level %s and pushgateway URL %s", logLevel, pushgatewayURL)

	e, err := euthanaiser.New(log, pushgatewayURL)
	if err != nil {
		log.Fatal(err)
	}

	e.Run(context.Background())
}

func envOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func setupLogger(level string) *log.Entry {
	ret := log.New()
	ret.SetFormatter(&log.JSONFormatter{})
	ret.SetOutput(os.Stdout)
	logLevel, err := log.ParseLevel(level)
	if err != nil {
		log.Fatalf("parsing log level: %v", err)
	}
	ret.SetLevel(logLevel)
	return ret.WithField("application", "euthanaisa")
}
