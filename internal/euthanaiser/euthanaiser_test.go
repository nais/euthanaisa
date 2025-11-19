package euthanaiser

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nais/euthanaisa/internal/client"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestEuthanaiser_ProcessResource(t *testing.T) {
	now := time.Now()
	type testCase struct {
		name            string
		annotations     map[string]string
		deleting        bool
		deleteErr       error
		expectDelete    bool
		expectDeleteErr bool
		expectKilled    bool
	}

	tests := []testCase{
		{
			name: "should delete expired resource",
			annotations: map[string]string{
				KillAfterAnnotation: now.Add(-1 * time.Hour).Format(time.RFC3339),
			},
			expectDelete: true,
			expectKilled: true,
		},
		{
			name: "should not delete future resource",
			annotations: map[string]string{
				KillAfterAnnotation: now.Add(1 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			name: "should skip already deleting",
			annotations: map[string]string{
				KillAfterAnnotation: now.Add(-1 * time.Hour).Format(time.RFC3339),
			},
			deleting: true,
		},
		{
			name: "should error on malformed annotation",
			annotations: map[string]string{
				KillAfterAnnotation: "invalid-time",
			},
			expectDeleteErr: true,
		},
		{
			name: "should handle delete not found",
			annotations: map[string]string{
				KillAfterAnnotation: now.Add(-1 * time.Hour).Format(time.RFC3339),
			},
			deleteErr:    errors.NewNotFound(schema.GroupResource{Group: "apps", Resource: "deployments"}, "name"),
			expectDelete: true,
		},
		{
			name: "should not delete if there is no kill-after annotation",
			annotations: map[string]string{
				"some-other-annotation": "value",
			},
			expectDelete: false,
			expectKilled: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRC := client.NewMockResourceClient(t)
			mockRC.On("GetResourceName").Return("applications")
			if tc.expectDelete {
				mockRC.On("GetResourceName").Return("applications")
				mockRC.On("Delete", mock.Anything, "ns", "name").Return(tc.deleteErr)
			}

			if tc.expectKilled {
				mockRC.On("GetResourceKind").Return("Deployment")
				mockRC.On("GetResourceGroup").Return("apps")
			}

			e := &euthanaiser{log: logrus.New()}
			u := &unstructured.Unstructured{}
			u.SetNamespace("ns")
			u.SetName("name")
			u.SetAnnotations(tc.annotations)
			if tc.deleting {
				ts := metav1.NewTime(now)
				u.SetDeletionTimestamp(&ts)
			}

			err := e.process(context.Background(), mockRC, u)

			if tc.expectDeleteErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.expectDelete {
				mockRC.AssertCalled(t, "GetResourceName")
				mockRC.AssertCalled(t, "Delete", mock.Anything, "ns", "name")
			} else {
				mockRC.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}

func TestEuthanaiser_Run(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(mockRC *client.MockResourceClient)
		expectedDelete bool
	}{
		{
			name: "should delete expired resource",
			setupMocks: func(mockRC *client.MockResourceClient) {
				res := &unstructured.Unstructured{}
				res.SetNamespace("ns")
				res.SetName("expired")
				res.SetAnnotations(map[string]string{
					KillAfterAnnotation: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				})

				mockRC.On("GetResourceKind").Return("Deployment")
				mockRC.On("List", mock.Anything, metav1.NamespaceAll, mock.AnythingOfType("client.ListOption")).Return([]*unstructured.Unstructured{res}, nil)
				mockRC.On("Delete", mock.Anything, "ns", "expired").Return(nil)
				mockRC.On("GetResourceName").Return("applications")
				mockRC.On("GetResourceGroup").Return("apps")
			},
			expectedDelete: true,
		},
		{
			name: "should skip future-dated resource",
			setupMocks: func(mockRC *client.MockResourceClient) {
				res := &unstructured.Unstructured{}
				res.SetNamespace("ns")
				res.SetName("future")
				res.SetAnnotations(map[string]string{
					KillAfterAnnotation: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
				})

				mockRC.On("GetResourceName").Return("applications")
				mockRC.On("List", mock.Anything, metav1.NamespaceAll, mock.AnythingOfType("client.ListOption")).Return([]*unstructured.Unstructured{res}, nil)
			},
			expectedDelete: false,
		},
		{
			name: "should skip resource with malformed annotation",
			setupMocks: func(mockRC *client.MockResourceClient) {
				res := &unstructured.Unstructured{}
				res.SetNamespace("ns")
				res.SetName("badtime")
				res.SetAnnotations(map[string]string{
					KillAfterAnnotation: "invalid-time-format",
				})

				mockRC.On("List", mock.Anything, metav1.NamespaceAll, mock.AnythingOfType("client.ListOption")).Return([]*unstructured.Unstructured{res}, nil)
				mockRC.On("GetResourceName").Return("applications")
				mockRC.On("GetResourceGroup").Return("apps")
			},
			expectedDelete: false,
		},
		{
			name: "should handle list error gracefully",
			setupMocks: func(mockRC *client.MockResourceClient) {
				mockRC.On("List", mock.Anything, metav1.NamespaceAll, mock.AnythingOfType("client.ListOption")).Return(nil, fmt.Errorf("list failed"))
				mockRC.On("GetResourceName").Return("applications")
				mockRC.On("GetResourceGroup").Return("apps")
			},
			expectedDelete: false,
		},
		{
			name: "should not delete if no kill-after annotation",
			setupMocks: func(mockRC *client.MockResourceClient) {
				res := &unstructured.Unstructured{}
				res.SetNamespace("ns")
				res.SetName("no-annotation")
				res.SetAnnotations(map[string]string{
					"some-other-annotation": "value",
				})

				mockRC.On("List", mock.Anything, metav1.NamespaceAll, mock.AnythingOfType("client.ListOption")).Return([]*unstructured.Unstructured{res}, nil)
				mockRC.On("GetResourceName").Return("applications")
			},
			expectedDelete: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRC := client.NewMockResourceClient(t)
			tt.setupMocks(mockRC)

			e := &euthanaiser{
				log:     logrus.New(),
				clients: []client.ResourceClient{mockRC},
			}

			e.Run(context.Background())

			if tt.expectedDelete {
				mockRC.AssertCalled(t, "Delete", mock.Anything, "ns", mock.Anything)
			} else {
				mockRC.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}

func TestEuthanaiser_Run_DelegatesToCorrectHandler(t *testing.T) {
	tests := []struct {
		name               string
		ownerRefs          []metav1.OwnerReference
		expectOwnerHandler bool
	}{
		{
			name:               "no owner reference, uses listing handler",
			ownerRefs:          nil,
			expectOwnerHandler: false,
		},
		{
			name: "has owner reference, uses owner handler",
			ownerRefs: []metav1.OwnerReference{
				{
					Kind: "OwnerKind",
					Name: "owner1",
				},
			},
			expectOwnerHandler: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := &unstructured.Unstructured{}
			resource.SetNamespace("ns")
			resource.SetName("resource1")
			resource.SetAnnotations(map[string]string{
				KillAfterAnnotation: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			})
			resource.SetOwnerReferences(tt.ownerRefs)

			ownerHandler := client.NewMockResourceClient(t)
			resourceHandler := client.NewMockResourceClient(t)

			resourceHandler.
				On("List", mock.Anything, metav1.NamespaceAll, mock.AnythingOfType("client.ListOption")).
				Return([]*unstructured.Unstructured{resource}, nil)

			// identities for rc handlers
			resourceHandler.On("GetResourceName").Return("resource").Maybe()
			resourceHandler.On("GetResourceKind").Return("ResourceKind").Maybe()
			resourceHandler.On("GetResourceGroup").Return("apps").Maybe()

			if tt.expectOwnerHandler {
				ownerHandler.On("GetResourceName").Return("owner").Maybe()
				ownerHandler.On("GetResourceKind").Return("OwnerKind").Maybe()
				ownerHandler.On("GetResourceGroup").Return("apps").Maybe()

				// Expect deletion of OWNER, not child
				ownerHandler.On("Delete", mock.Anything, "ns", "owner1").Return(nil).Once()
			}

			if !tt.expectOwnerHandler {
				// Expect deletion of CHILD itself
				resourceHandler.On("Delete", mock.Anything, "ns", "resource1").Return(nil).Once()
			}

			e := &euthanaiser{
				log: logrus.New(),
				clients: []client.ResourceClient{
					resourceHandler,
				},
				resourceHandlersByKind: client.HandlerByKind{
					"OwnerKind": ownerHandler,
				},
			}

			e.Run(context.Background())

			if tt.expectOwnerHandler {
				ownerHandler.AssertCalled(t, "Delete", mock.Anything, "ns", "owner1")
				resourceHandler.AssertNotCalled(t, "Delete", mock.Anything, "ns", "resource1")
			} else {
				resourceHandler.AssertCalled(t, "Delete", mock.Anything, "ns", "resource1")
				ownerHandler.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}
