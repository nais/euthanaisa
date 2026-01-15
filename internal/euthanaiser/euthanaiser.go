// Package euthanaiser contains the core logic for deleting expired Kubernetes resources.
package euthanaiser

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/nais/euthanaisa/internal/client"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const KillAfterLabel = "euthanaisa.nais.io/kill-after"

var (
	deleted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "euthanaisa_deleted_total",
			Help: "Number of resources deleted",
		},
		[]string{"resource"},
	)
	errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "euthanaisa_errors_total",
			Help: "Number of errors encountered",
		},
		[]string{"resource"},
	)
)

func init() {
	prometheus.MustRegister(deleted, errorsTotal)
}

type Euthanaiser struct {
	clients []client.ResourceClient
	now     func() time.Time
}

func New(clients []client.ResourceClient) *Euthanaiser {
	return &Euthanaiser{
		clients: clients,
		now:     time.Now,
	}
}

func (e *Euthanaiser) Run(ctx context.Context) {
	for _, rc := range e.clients {
		e.processResources(ctx, rc)
	}
	slog.Info("finished processing all resources")
}

func (e *Euthanaiser) processResources(ctx context.Context, rc client.ResourceClient) {
	list, err := rc.List(ctx, KillAfterLabel)
	if err != nil {
		slog.Error("listing resources", "resource", rc.Name(), "error", err)
		errorsTotal.WithLabelValues(rc.Name()).Inc()
		return
	}

	slog.Debug("scanned resources", "resource", rc.Name(), "count", len(list))

	for _, item := range list {
		if err := e.process(ctx, rc, item); err != nil {
			slog.Error("processing resource", "resource", rc.Name(), "error", err)
			errorsTotal.WithLabelValues(rc.Name()).Inc()
		}
	}
}

func (e *Euthanaiser) process(ctx context.Context, rc client.ResourceClient, u *unstructured.Unstructured) error {
	if !e.shouldDelete(u) {
		return nil
	}

	if err := rc.Delete(ctx, u.GetNamespace(), u.GetName()); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting %s/%s: %w", u.GetNamespace(), u.GetName(), err)
	}

	slog.Info("deleted", "namespace", u.GetNamespace(), "name", u.GetName(), "kind", u.GetKind())
	deleted.WithLabelValues(rc.Name()).Inc()
	return nil
}

func (e *Euthanaiser) shouldDelete(u *unstructured.Unstructured) bool {
	if u.GetDeletionTimestamp() != nil {
		return false
	}

	killAfterStr := u.GetLabels()[KillAfterLabel]
	if killAfterStr == "" {
		return false
	}

	killAfterUnix, err := strconv.ParseInt(killAfterStr, 10, 64)
	if err != nil {
		slog.Warn("invalid kill-after label", "value", killAfterStr, "error", err)
		return false
	}

	return time.Unix(killAfterUnix, 0).Before(e.now())
}
