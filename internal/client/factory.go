package client

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/nais/euthanaisa/internal/config"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
)

type Factory struct {
	dyn dynamic.Interface
	log logrus.FieldLogger
}

func NewFactory(dyn dynamic.Interface, log logrus.FieldLogger) *Factory {
	return &Factory{
		dyn: dyn,
		log: log,
	}
}

func (f *Factory) BuildClients(resourceConfigs []config.ResourceConfig) ([]ResourceClient, error) {
	var handlers []ResourceClient

	for _, r := range resourceConfigs {
		handler, err := newResourceHandler(f.dyn, r)
		if err != nil {
			return nil, fmt.Errorf("building resource handler for %s: %w", r.Resource, err)
		}
		f.log.WithFields(logrus.Fields{
			"resource": r.Resource,
			"group":    r.Group,
			"version":  r.Version,
		}).Info("created resource handler")
		handlers = append(handlers, handler)
	}

	return handlers, nil
}

func newResourceHandler(dyn dynamic.Interface, cfg config.ResourceConfig) (ResourceClient, error) {
	gvr := schema.GroupVersionResource{Group: cfg.Group, Version: cfg.Version, Resource: cfg.Resource}
	return &resourceHandler{
		client:   dyn.Resource(gvr),
		resource: cfg,
	}, nil
}
