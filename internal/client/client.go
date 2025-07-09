package client

import (
	"context"

	"github.com/nais/euthanaisa/internal/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

type ResourceClient interface {
	List(ctx context.Context, namespace string, opts ...ListOption) ([]*unstructured.Unstructured, error)
	Delete(ctx context.Context, namespace, name string) error
	GetResourceName() string
	GetResourceKind() string
	GetResourceGroup() string
}

type resourceClientImpl struct {
	client   dynamic.NamespaceableResourceInterface
	resource config.ResourceConfig
}

func (r *resourceClientImpl) List(ctx context.Context, namespace string, opts ...ListOption) ([]*unstructured.Unstructured, error) {
	listOptions := metav1.ListOptions{}
	for _, opt := range opts {
		opt(&listOptions)
	}

	list, err := r.client.Namespace(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	result := make([]*unstructured.Unstructured, 0, len(list.Items))
	for i := range list.Items {
		result = append(result, &list.Items[i])
	}
	return result, nil
}

func (r *resourceClientImpl) Delete(ctx context.Context, namespace, name string) error {
	return r.client.Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (r *resourceClientImpl) GetResourceName() string {
	return r.resource.Resource
}

func (r *resourceClientImpl) GetResourceKind() string {
	return r.resource.Kind
}

func (r *resourceClientImpl) GetResourceGroup() string {
	return r.resource.Group
}

type ListOption func(*metav1.ListOptions)

func WithLabelSelector(selector string) ListOption {
	return func(opt *metav1.ListOptions) {
		opt.LabelSelector = selector
	}
}
