package euthanaiser

import (
	"context"
	"fmt"
	"time"

	"github.com/nais/euthanaisa/internal/client"
	"github.com/nais/euthanaisa/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// KillAfterAnnotation is the key used to mark when a resource should be deleted by euthanaisa.
	KillAfterAnnotation = "euthanaisa.nais.io/kill-after"
)

type euthanaiser struct {
	ownerClients           []client.ResourceClient
	resourceHandlersByKind client.HandlerByKind
	pusher                 *push.Pusher
	log                    logrus.FieldLogger
}

func New(ownerClients []client.ResourceClient, resourceHandlersByKind client.HandlerByKind, pusher *push.Pusher, log logrus.FieldLogger) *euthanaiser {
	return &euthanaiser{
		ownerClients:           ownerClients,
		resourceHandlersByKind: resourceHandlersByKind,
		pusher:                 pusher,
		log:                    log,
	}
}

func (e *euthanaiser) Run(ctx context.Context) {
	defer e.pushMetrics(ctx)

	for _, rc := range e.ownerClients {
		e.listAndProcessResources(ctx, rc)
	}

	e.log.Info("finished scanning and processing all configured resources")
}

func (e *euthanaiser) listAndProcessResources(ctx context.Context, rc client.ResourceClient) {
	list, err := rc.List(ctx, metav1.NamespaceAll)
	if err != nil {
		e.log.WithError(err).WithField("resource", rc.GetResourceName()).Error("listing resources")
		metrics.ResourceErrors.WithLabelValues(rc.GetResourceGroup(), rc.GetResourceName()).Inc()
		return
	}

	metrics.ResourcesScannedTotal.WithLabelValues(rc.GetResourceName()).Add(float64(len(list)))

	for _, item := range list {
		handler := e.getResourceHandlerForOwnedResource(item, rc)
		if err := e.process(ctx, handler, item); err != nil {
			e.log.WithError(err).WithField("resource", handler.GetResourceName()).Error("processing resource")
			metrics.ResourceErrors.WithLabelValues(handler.GetResourceGroup(), handler.GetResourceName()).Inc()
		}
	}
}

func (e *euthanaiser) getResourceHandlerForOwnedResource(item *unstructured.Unstructured, defaultRC client.ResourceClient) client.ResourceClient {
	for _, owner := range item.GetOwnerReferences() {
		if handler, ok := e.resourceHandlersByKind.Get(owner.Kind); ok {
			return handler
		}
	}
	return defaultRC
}

func (e *euthanaiser) process(ctx context.Context, rc client.ResourceClient, u *unstructured.Unstructured) error {
	shouldKill, err := shouldBeKilled(u, rc.GetResourceName())
	if err != nil {
		return fmt.Errorf("checking if resource should be killed: %w", err)
	}
	if !shouldKill {
		return nil
	}

	timer := prometheus.NewTimer(metrics.ResourceDeleteDuration.WithLabelValues(rc.GetResourceName()))
	defer timer.ObserveDuration()

	err = rc.Delete(ctx, u.GetNamespace(), u.GetName())
	if err != nil {
		if errors.IsNotFound(err) {
			e.log.WithFields(logrus.Fields{
				"namespace": u.GetNamespace(),
				"name":      u.GetName(),
				"resource":  rc.GetResourceName(),
			}).Debug("resource already deleted")
			return nil // already deleted
		}
		return fmt.Errorf("deleting resource %s/%s: %w", u.GetNamespace(), u.GetName(), err)
	}
	e.log.WithFields(logrus.Fields{
		"namespace": u.GetNamespace(),
		"name":      u.GetName(),
		"resource":  u.GetKind(),
		"owned-by":  rc.GetResourceKind(),
	}).Debug("deleted resource")
	metrics.ResourceKilled.WithLabelValues(rc.GetResourceGroup(), rc.GetResourceName()).Inc()
	return nil
}

func shouldBeKilled(u *unstructured.Unstructured, rCResourceName string) (bool, error) {
	if u.GetDeletionTimestamp() != nil {
		return false, nil // already deleting
	}

	killAfterStr := u.GetAnnotations()[KillAfterAnnotation]
	if killAfterStr == "" {
		return false, nil // no annotation
	}

	metrics.ResourcesKillableTotal.WithLabelValues(rCResourceName, u.GetNamespace()).Inc()

	killAfter, err := time.Parse(time.RFC3339, killAfterStr)
	if err != nil {
		return false, fmt.Errorf("parsing killAfter annotation: %w", err)
	}

	return killAfter.Before(time.Now()), nil
}

func (e *euthanaiser) pushMetrics(ctx context.Context) {
	if e.pusher == nil {
		e.log.Debug("metrics pusher disabled; skipping push")
		return
	}

	if err := e.pusher.AddContext(ctx); err != nil {
		e.log.WithError(err).Error("pushing metrics")
	} else {
		e.log.Info("pushed metrics to Pushgateway")
	}
}
