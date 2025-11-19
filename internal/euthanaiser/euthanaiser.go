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
	// LabelSelectorEnabledResources is the label selector used to filter resources that should be processed by euthanaisa.
	LabelSelectorEnabledResources = "euthanaisa.nais.io/enabled=true"
)

type euthanaiser struct {
	clients                []client.ResourceClient
	resourceHandlersByKind client.HandlerByKind
	pusher                 *push.Pusher
	log                    logrus.FieldLogger
}

func New(clients []client.ResourceClient, resourceHandlersByKind client.HandlerByKind, pusher *push.Pusher, log logrus.FieldLogger) *euthanaiser {
	return &euthanaiser{
		clients:                clients,
		resourceHandlersByKind: resourceHandlersByKind,
		pusher:                 pusher,
		log:                    log,
	}
}

func (e *euthanaiser) Run(ctx context.Context) {
	defer e.pushMetrics(ctx)

	for _, rc := range e.clients {
		e.listAndProcessResources(ctx, rc)
	}

	e.log.Info("finished scanning and processing all configured resources")
}

func (e *euthanaiser) listAndProcessResources(ctx context.Context, rc client.ResourceClient) {
	list, err := rc.List(ctx, metav1.NamespaceAll, client.WithLabelSelector(LabelSelectorEnabledResources))
	if err != nil {
		e.log.WithError(err).WithField("resource", rc.GetResourceName()).Error("listing resources")
		metrics.ResourceErrors.WithLabelValues(rc.GetResourceGroup(), rc.GetResourceName()).Inc()
		return
	}

	e.log.WithFields(logrus.Fields{
		"resource": rc.GetResourceName(),
		"count":    len(list),
	}).Debug("scanned resources")

	metrics.ResourcesScannedTotal.WithLabelValues(rc.GetResourceName()).Add(float64(len(list)))

	for _, item := range list {
		if len(item.GetOwnerReferences()) > 0 {
			if err = e.processChildOwnedResource(ctx, item); err != nil {
				e.log.WithError(err).WithFields(logrus.Fields{
					"namespace": item.GetNamespace(),
					"name":      item.GetName(),
					"kind":      item.GetKind(),
				}).Error("processing owned resource for owner deletion")
			}
			continue
		}

		// Root resource (no ownerReferences): delete based on its own kill-after annotation.
		if err := e.process(ctx, rc, item); err != nil {
			e.log.WithError(err).WithField("resource", rc.GetResourceName()).Error("processing resource")
			metrics.ResourceErrors.WithLabelValues(rc.GetResourceGroup(), rc.GetResourceName()).Inc()
		}
	}
}

func (e *euthanaiser) processChildOwnedResource(ctx context.Context, child *unstructured.Unstructured) error {
	ownerHandler, ownerName, ok := e.getConfiguredOwnerHandler(child)
	if !ok {
		e.log.WithFields(logrus.Fields{
			"namespace": child.GetNamespace(),
			"name":      child.GetName(),
			"kind":      child.GetKind(),
		}).Debug("owned resource has no configured owner handler; skipping")
		return nil
	}

	// Decide based on the CHILD's annotation whether we should delete the owner.
	shouldKill, err := shouldBeKilled(child, ownerHandler.GetResourceName(), e.log)
	if err != nil {
		return fmt.Errorf("checking if child resource should trigger owner deletion: %w", err)
	}
	if !shouldKill {
		return nil
	}

	timer := prometheus.NewTimer(metrics.ResourceDeleteDuration.WithLabelValues(ownerHandler.GetResourceName()))
	defer timer.ObserveDuration()

	err = ownerHandler.Delete(ctx, child.GetNamespace(), ownerName)
	if err != nil {
		if errors.IsNotFound(err) {
			e.log.WithFields(logrus.Fields{
				"namespace": child.GetNamespace(),
				"name":      ownerName,
				"kind":      ownerHandler.GetResourceKind(),
			}).Debug("owner resource already deleted")
			return nil
		}

		metrics.ResourceErrors.WithLabelValues(ownerHandler.GetResourceGroup(), ownerHandler.GetResourceName()).Inc()
		return fmt.Errorf("deleting owner resource %s/%s: %w", child.GetNamespace(), ownerName, err)
	}

	e.log.WithFields(logrus.Fields{
		"child-namespace": child.GetNamespace(),
		"child-name":      child.GetName(),
		"child-kind":      child.GetKind(),
		"owner-namespace": child.GetNamespace(),
		"owner-name":      ownerName,
		"owner-kind":      ownerHandler.GetResourceKind(),
		"owner-resource":  ownerHandler.GetResourceName(),
	}).Info("deleted owner resource due to child kill-after annotation")

	metrics.ResourceKilled.WithLabelValues(ownerHandler.GetResourceGroup(), ownerHandler.GetResourceName()).Inc()
	return nil
}

// getConfiguredOwnerHandler returns the first owner whose Kind exists in the configured HandlerByKind map.
// This implements: "Only delete the owner IF it’s in your configured resource list".
func (e *euthanaiser) getConfiguredOwnerHandler(child *unstructured.Unstructured) (client.ResourceClient, string, bool) {
	for _, owner := range child.GetOwnerReferences() {
		if handler, ok := e.resourceHandlersByKind.Get(owner.Kind); ok {
			return handler, owner.Name, true
		}
	}
	return nil, "", false
}

func (e *euthanaiser) process(ctx context.Context, rc client.ResourceClient, u *unstructured.Unstructured) error {
	shouldKill, err := shouldBeKilled(u, rc.GetResourceName(), e.log)
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
				"kind":      u.GetKind(),
				"resource":  rc.GetResourceName(),
			}).Debug("resource already deleted")
			return nil // already deleted
		}
		return fmt.Errorf("deleting resource %s/%s: %w", u.GetNamespace(), u.GetName(), err)
	}
	e.log.WithFields(logrus.Fields{
		"namespace":      u.GetNamespace(),
		"name":           u.GetName(),
		"kind":           u.GetKind(),
		"owned":          len(u.GetOwnerReferences()) > 0,
		"owner-resource": rc.GetResourceKind(),
	}).Info("deleted resource")

	metrics.ResourceKilled.WithLabelValues(rc.GetResourceGroup(), rc.GetResourceName()).Inc()
	return nil
}

func shouldBeKilled(u *unstructured.Unstructured, rCResourceName string, log logrus.FieldLogger) (bool, error) {
	if u.GetDeletionTimestamp() != nil {
		return false, nil // already deleting
	}

	killAfterStr := u.GetAnnotations()[KillAfterAnnotation]
	if killAfterStr == "" {
		return false, nil // no annotation
	}

	log.WithFields(logrus.Fields{
		"namespace":      u.GetNamespace(),
		"name":           u.GetName(),
		"owned":          len(u.GetOwnerReferences()) > 0,
		"owner-resource": rCResourceName,
		"kind":           u.GetKind(),
		"killAfter":      killAfterStr,
	}).Debug("found resource with kill-after annotation")

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
