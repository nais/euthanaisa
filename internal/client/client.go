package client

import (
	"context"
	"fmt"

	"github.com/nais/euthanaisa/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type ResourceClient interface {
	List(ctx context.Context, namespace string) ([]*unstructured.Unstructured, error)
	Delete(ctx context.Context, namespace, name string) error
	GetResourceName() string
	GetResourceKind() string
	IncKilledMetric()
	IncErrorMetric()
}

type resourceHandler struct {
	client   dynamic.NamespaceableResourceInterface
	resource config.ResourceConfig

	metricKilled prometheus.Counter
	metricError  prometheus.Counter
}

func (r *resourceHandler) List(ctx context.Context, namespace string) ([]*unstructured.Unstructured, error) {
	list, err := r.client.Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	result := make([]*unstructured.Unstructured, 0, len(list.Items))
	for i := range list.Items {
		result = append(result, &list.Items[i])
	}
	return result, nil
}

func (r *resourceHandler) Delete(ctx context.Context, namespace, name string) error {
	return r.client.Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (r *resourceHandler) GetResourceName() string {
	return r.resource.Resource
}

func (r *resourceHandler) GetResourceKind() string {
	return r.resource.Name
}

func (r *resourceHandler) IncKilledMetric() {
	r.metricKilled.Inc()
}

func (r *resourceHandler) IncErrorMetric() {
	r.metricError.Inc()
}

func LoadResourceClients(resources []config.ResourceConfig, kc *rest.Config, registry *prometheus.Registry, log logrus.FieldLogger) ([]ResourceClient, error) {
	dyn, err := dynamic.NewForConfig(kc)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	var handlers []ResourceClient
	for _, r := range resources {
		gvr := schema.GroupVersionResource{Group: r.Group, Version: r.Version, Resource: r.Resource}
		metricKilled := prometheus.NewCounter(prometheus.CounterOpts{
			Name: fmt.Sprintf("euthanaisa_%s_killed", r.Resource),
			Help: fmt.Sprintf("Number of %s %s killed by euthanaisa", r.Group, r.Resource),
		})
		metricErrors := prometheus.NewCounter(prometheus.CounterOpts{
			Name: fmt.Sprintf("euthanaisa_%s_errors", r.Resource),
			Help: fmt.Sprintf("Number of errors encountered while processing %s %s", r.Group, r.Resource),
		})

		handler := &resourceHandler{
			client:       dyn.Resource(gvr),
			metricKilled: metricKilled,
			metricError:  metricErrors,
			resource:     r,
		}
		registry.MustRegister(metricKilled, metricErrors)
		log.WithField("resource", r.Resource).Infof("registered client for %s/%s", r.Group, r.Resource)

		handlers = append(handlers, handler)
	}
	return handlers, nil
}
