package client

import (
	"context"

	"github.com/nais/euthanaisa/internal/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

const LabelSelectorEnabledResources = "euthanaisa.nais.io/enabled=true"

type ResourceClient interface {
	List(ctx context.Context, namespace string) ([]*unstructured.Unstructured, error)
	Delete(ctx context.Context, namespace, name string) error
	GetResourceName() string
	GetResourceKind() string
	GetResourceGroup() string
}

type resourceClientImpl struct {
	client   dynamic.NamespaceableResourceInterface
	resource config.ResourceConfig
}

func (r *resourceClientImpl) List(ctx context.Context, namespace string) ([]*unstructured.Unstructured, error) {
	list, err := r.client.Namespace(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: LabelSelectorEnabledResources,
	})
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
