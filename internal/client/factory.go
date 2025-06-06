package client

import (
	"fmt"

	"github.com/nais/euthanaisa/internal/config"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type factory struct {
	dyn dynamic.Interface
	log logrus.FieldLogger
}

func NewFactory(dyn dynamic.Interface, log logrus.FieldLogger) *factory {
	return &factory{
		dyn: dyn,
		log: log,
	}
}

type HandlerByKind map[string]ResourceClient

func (h HandlerByKind) Get(kind string) (ResourceClient, bool) {
	return h[kind], h[kind] != nil
}

func (f *factory) BuildClients(resourceConfigs []config.ResourceConfig) ([]ResourceClient, HandlerByKind, error) {
	var ownerClients []ResourceClient
	handlerByKind := make(HandlerByKind)

	for _, r := range resourceConfigs {
		handler, err := newResourceHandler(f.dyn, r)
		if err != nil {
			return nil, nil, fmt.Errorf("building resource handler for %s: %w", r.Resource, err)
		}

		f.log.WithFields(logrus.Fields{
			"resource": r.Resource,
			"group":    r.Group,
			"version":  r.Version,
			"kind":     r.Kind,
		}).Info("created resource handler")

		if r.Kind != "" {
			handlerByKind[r.Kind] = handler
		} else {
			f.log.Warnf("resource client %s has empty kind; skipping handlerByKind map entry", r.Resource)
		}

		// Only include as an "owner" if it owns other resources
		if len(r.OwnedBy) > 0 {
			ownerClients = append(ownerClients, handler)
		}
	}

	return ownerClients, handlerByKind, nil
}

func newResourceHandler(dyn dynamic.Interface, cfg config.ResourceConfig) (ResourceClient, error) {
	gvr := schema.GroupVersionResource{Group: cfg.Group, Version: cfg.Version, Resource: cfg.Resource}
	return &resourceHandler{
		client:   dyn.Resource(gvr),
		resource: cfg,
	}, nil
}
