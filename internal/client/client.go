package client

import (
	"context"

	"github.com/nais/euthanaisa/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

type ResourceClient interface {
	List(ctx context.Context, namespace string) ([]*unstructured.Unstructured, error)
	Delete(ctx context.Context, namespace, name string) error
	GetResourceName() string
	GetResourceKind() string
	ShouldProcess() bool
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

func (r *resourceHandler) ShouldProcess() bool {
	return len(r.resource.OwnedBy) > 0
}

func (r *resourceHandler) IncKilledMetric() {
	r.metricKilled.Inc()
}

func (r *resourceHandler) IncErrorMetric() {
	r.metricError.Inc()
}
