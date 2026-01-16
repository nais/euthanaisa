package euthanaiser

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nais/euthanaisa/internal/client"
	"github.com/stretchr/testify/assert"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type fakeClient struct {
	resources    []*unstructured.Unstructured
	listErr      error
	deleteErr    error
	deletedNames []string
}

func (f *fakeClient) List(_ context.Context, _ string) ([]*unstructured.Unstructured, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.resources, nil
}

func (f *fakeClient) Delete(_ context.Context, _, name string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deletedNames = append(f.deletedNames, name)
	return nil
}

func (f *fakeClient) Name() string { return "test" }

var _ client.ResourceClient = (*fakeClient)(nil)

func TestShouldDelete(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	e := &Euthanaiser{now: func() time.Time { return fixedTime }}

	tests := []struct {
		name     string
		labels   map[string]string
		deleting bool
		want     bool
	}{
		{"expired", map[string]string{KillAfterLabel: "1705316400"}, false, true}, // 11:00 UTC
		{"future", map[string]string{KillAfterLabel: "1705323600"}, false, false}, // 13:00 UTC
		{"no label", map[string]string{"other": "value"}, false, false},
		{"empty labels", nil, false, false},
		{"already deleting", map[string]string{KillAfterLabel: "1705316400"}, true, false},
		{"invalid timestamp", map[string]string{KillAfterLabel: "not-a-timestamp"}, false, false},
		{"exactly now", map[string]string{KillAfterLabel: "1705320000"}, false, false}, // 12:00 UTC
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &unstructured.Unstructured{}
			u.SetLabels(tt.labels)
			if tt.deleting {
				ts := metav1.NewTime(fixedTime)
				u.SetDeletionTimestamp(&ts)
			}
			assert.Equal(t, tt.want, e.shouldDelete(u))
		})
	}
}

func TestRun(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	t.Run("deletes expired", func(t *testing.T) {
		fake := &fakeClient{resources: []*unstructured.Unstructured{
			newResource("ns", "expired", map[string]string{KillAfterLabel: "1705316400"}),
		}}
		e := &Euthanaiser{clients: []client.ResourceClient{fake}, now: func() time.Time { return fixedTime }}
		e.Run(context.Background())
		assert.Equal(t, []string{"expired"}, fake.deletedNames)
	})

	t.Run("skips future", func(t *testing.T) {
		fake := &fakeClient{resources: []*unstructured.Unstructured{
			newResource("ns", "future", map[string]string{KillAfterLabel: "1705323600"}),
		}}
		e := &Euthanaiser{clients: []client.ResourceClient{fake}, now: func() time.Time { return fixedTime }}
		e.Run(context.Background())
		assert.Empty(t, fake.deletedNames)
	})

	t.Run("handles list error", func(t *testing.T) {
		fake := &fakeClient{listErr: errors.New("api error")}
		e := &Euthanaiser{clients: []client.ResourceClient{fake}, now: func() time.Time { return fixedTime }}
		e.Run(context.Background())
	})

	t.Run("handles NotFound", func(t *testing.T) {
		fake := &fakeClient{
			resources: []*unstructured.Unstructured{
				newResource("ns", "gone", map[string]string{KillAfterLabel: "1705316400"}),
			},
			deleteErr: k8serrors.NewNotFound(schema.GroupResource{}, "gone"),
		}
		e := &Euthanaiser{clients: []client.ResourceClient{fake}, now: func() time.Time { return fixedTime }}
		e.Run(context.Background())
	})
}

func newResource(namespace, name string, labels map[string]string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetNamespace(namespace)
	u.SetName(name)
	u.SetLabels(labels)
	return u
}
