package euthanaiser

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func Test_ShouldBeKilled(t *testing.T) {
	tests := []struct {
		name              string
		annotations       map[string]string
		deletionTimestamp *metav1.Time
		want              bool
		wantErr           bool
	}{
		{
			name:        "no annotation",
			annotations: map[string]string{},
			want:        false,
		},
		{
			name:        "invalid timestamp",
			annotations: map[string]string{KillAfterAnnotation: "invalid"},
			want:        false,
			wantErr:     true,
		},
		{
			name:        "future timestamp",
			annotations: map[string]string{KillAfterAnnotation: time.Now().Add(1 * time.Hour).Format(time.RFC3339)},
			want:        false,
		},
		{
			name:        "past timestamp",
			annotations: map[string]string{KillAfterAnnotation: time.Now().Add(-1 * time.Hour).Format(time.RFC3339)},
			want:        true,
		},
		{
			name:              "resource already deleting",
			annotations:       map[string]string{KillAfterAnnotation: time.Now().Add(-1 * time.Hour).Format(time.RFC3339)},
			deletionTimestamp: &metav1.Time{Time: time.Now()},
			want:              false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &unstructured.Unstructured{}
			obj.SetAnnotations(tt.annotations)
			if tt.deletionTimestamp != nil {
				obj.SetDeletionTimestamp(tt.deletionTimestamp)
			}

			got, err := shouldBeKilled(obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("shouldBeKilled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("shouldBeKilled() = %v, want %v", got, tt.want)
			}
		})
	}
}
