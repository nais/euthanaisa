// Package client provides a Kubernetes dynamic client wrapper for listing and deleting resources.
package client

import (
	"context"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type Resource struct {
	Group    string
	Version  string
	Resource string
}

type ResourceClient interface {
	List(ctx context.Context, labelSelector string) ([]*unstructured.Unstructured, error)
	Delete(ctx context.Context, namespace, name string) error
	Name() string
}

type resourceClient struct {
	client dynamic.NamespaceableResourceInterface
	name   string
}

func (r *resourceClient) List(ctx context.Context, labelSelector string) ([]*unstructured.Unstructured, error) {
	list, err := r.client.Namespace(metav1.NamespaceAll).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
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

func (r *resourceClient) Delete(ctx context.Context, namespace, name string) error {
	return r.client.Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (r *resourceClient) Name() string {
	return r.name
}

func BuildClients(dyn dynamic.Interface, resources []Resource) []ResourceClient {
	clients := make([]ResourceClient, 0, len(resources))
	for _, r := range resources {
		gvr := schema.GroupVersionResource{Group: r.Group, Version: r.Version, Resource: r.Resource}
		clients = append(clients, &resourceClient{
			client: dyn.Resource(gvr),
			name:   r.Resource,
		})
		slog.Debug("registered resource", "resource", r.Resource)
	}
	return clients
}
