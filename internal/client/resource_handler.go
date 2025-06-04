package client

import (
	"fmt"

	"github.com/nais/euthanaisa/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func NewResourceHandler(
	dyn dynamic.Interface,
	cfg config.ResourceConfig,
	registry *prometheus.Registry,
) (ResourceClient, error) {
	gvr := schema.GroupVersionResource{Group: cfg.Group, Version: cfg.Version, Resource: cfg.Resource}

	metricKilled := prometheus.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("euthanaisa_%s_killed", cfg.Resource),
		Help: fmt.Sprintf("Number of %s %s killed by euthanaisa", cfg.Group, cfg.Resource),
	})
	metricErrors := prometheus.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("euthanaisa_%s_errors", cfg.Resource),
		Help: fmt.Sprintf("Number of errors encountered while processing %s %s", cfg.Group, cfg.Resource),
	})

	if err := registry.Register(metricKilled); err != nil {
		return nil, fmt.Errorf("registering killed metric: %w", err)
	}
	if err := registry.Register(metricErrors); err != nil {
		return nil, fmt.Errorf("registering error metric: %w", err)
	}

	return &resourceHandler{
		client:       dyn.Resource(gvr),
		resource:     cfg,
		metricKilled: metricKilled,
		metricError:  metricErrors,
	}, nil
}
