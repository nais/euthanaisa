package client

import (
	"fmt"

	"github.com/nais/euthanaisa/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
)

type Factory struct {
	dyn      dynamic.Interface
	registry *prometheus.Registry
	log      logrus.FieldLogger
}

func NewFactory(dyn dynamic.Interface, registry *prometheus.Registry, log logrus.FieldLogger) *Factory {
	return &Factory{
		dyn:      dyn,
		registry: registry,
		log:      log,
	}
}

func (f *Factory) BuildClients(resourceConfigs []config.ResourceConfig) ([]ResourceClient, error) {
	var handlers []ResourceClient

	for _, r := range resourceConfigs {
		handler, err := NewResourceHandler(f.dyn, r, f.registry)
		if err != nil {
			return nil, fmt.Errorf("building resource handler for %s: %w", r.Resource, err)
		}
		f.log.WithField("resource", r.Resource).Infof("registered client for %s/%s", r.Group, r.Resource)
		handlers = append(handlers, handler)
	}

	return handlers, nil
}
