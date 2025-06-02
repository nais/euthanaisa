package euthanaiser

import (
	"context"
	"fmt"
	"time"

	"github.com/nais/euthanaisa/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const (
	// KillAfterAnnotation Annotation used to specify when a resource should be deleted by euthanaisa.
	KillAfterAnnotation = "euthanaisa.nais.io/kill-after"
)

type euthanaiser struct {
	resourcesClients map[string]resourceHandler
	pusher           *push.Pusher
	log              logrus.FieldLogger
}

type resourceHandler struct {
	client       dynamic.NamespaceableResourceInterface
	metricKilled prometheus.Counter
	metricError  prometheus.Counter
	resource     config.Resource
}

func New(cfg *config.Config, log logrus.FieldLogger) (*euthanaiser, error) {
	kc, err := config.Kubeconfig()
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig: %w", err)
	}

	registry := prometheus.NewRegistry()

	handlers, err := loadResourceHandlers(cfg, kc, registry, log)
	if err != nil {
		return nil, fmt.Errorf("loading resource handlers: %w", err)
	}

	var pusher *push.Pusher
	if cfg.PushgatewayURL != "" {
		pusher = push.New(cfg.PushgatewayURL, "euthanaisa").Gatherer(registry)
	} else {
		log.Infof("Pushgateway URL not set; metrics will not be pushed")
	}

	return &euthanaiser{
		resourcesClients: handlers,
		pusher:           pusher,
		log:              log,
	}, nil
}

func (e *euthanaiser) Run(ctx context.Context) {
	defer e.pushMetrics(ctx)

	for _, rc := range e.resourcesClients {
		list, err := rc.client.Namespace("").List(ctx, metav1.ListOptions{
			// Use a field selector to only list resources that have the euthanaisa annotation,
			// effectively filtering out resources that should not be processed.
			// LabelSelector: "euthanaisa.nais.io/kill: true",
		})
		if err != nil {
			e.log.WithError(err).WithField("resource", rc.resource.GetResourceName()).Error("listing resources")
			rc.metricError.Inc()
			continue
		}

		for _, item := range list.Items {
			handler := e.getResourceHandlerForOwnedResource(item, rc)
			if err := e.process(ctx, handler, &item); err != nil {
				e.log.WithError(err).WithField("resource", handler.resource.GetResourceName()).Error("processing resource")
				rc.metricError.Inc()
			}
		}
	}

	e.log.Info("finished processing all configured resources")
}

func (e *euthanaiser) getResourceHandlerForOwnedResource(item unstructured.Unstructured, defaultHandler resourceHandler) resourceHandler {
	for _, owner := range item.GetOwnerReferences() {
		switch owner.Kind {
		case "Application":
			if handler, ok := e.resourcesClients["applications"]; ok {
				return handler
			}
		case "Naisjob":
			if handler, ok := e.resourcesClients["naisjobs"]; ok {
				return handler
			}
		}
	}
	// fallback to the default handler
	return defaultHandler
}

func (e *euthanaiser) process(ctx context.Context, rc resourceHandler, u *unstructured.Unstructured) error {
	shouldKill, err := shouldBeKilled(u)
	if err != nil {
		return fmt.Errorf("checking if resource should be killed: %w", err)
	}
	if !shouldKill {
		return nil
	}

	// Delete resource
	err = rc.client.Namespace(u.GetNamespace()).Delete(ctx, u.GetName(), metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil // already deleted
		}
		return fmt.Errorf("deleting resource %s/%s: %w", u.GetNamespace(), u.GetName(), err)
	}
	e.log.WithFields(logrus.Fields{
		"namespace": u.GetNamespace(),
		"name":      u.GetName(),
		"resource":  rc.resource.GetResourceName(),
	}).Debugf("Deleted resource")
	rc.metricKilled.Inc()
	return nil
}

func shouldBeKilled(u *unstructured.Unstructured) (bool, error) {
	if u.GetDeletionTimestamp() != nil {
		return false, nil // already deleting
	}

	killAfterStr := u.GetAnnotations()[KillAfterAnnotation]
	if killAfterStr == "" {
		return false, nil // no annotation
	}

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
		e.log.Info("pushed metrics to prometheus")
	}
}

func loadResourceHandlers(cfg *config.Config, kc *rest.Config, registry *prometheus.Registry, log logrus.FieldLogger) (map[string]resourceHandler, error) {
	dyn, err := dynamic.NewForConfig(kc)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	handlers := make(map[string]resourceHandler)
	for _, resource := range cfg.Resources {
		gvr := schema.GroupVersionResource{Group: resource.GetGroup(), Version: resource.GetVersion(), Resource: resource.GetResourceName()}
		metricKilled := prometheus.NewCounter(prometheus.CounterOpts{
			Name: fmt.Sprintf("euthanaisa_%s_killed", resource.GetResourceName()),
			Help: fmt.Sprintf("Number of %s %s killed by euthanaisa", resource.GetGroup(), resource.GetResourceName()),
		})
		metricErrors := prometheus.NewCounter(prometheus.CounterOpts{
			Name: fmt.Sprintf("euthanaisa_%s_errors", resource.GetResourceName()),
			Help: fmt.Sprintf("Number of errors encountered while processing %s %s", resource.GetGroup(), resource.GetResourceName()),
		})

		handlers[resource.GetResourceName()] = resourceHandler{
			client:       dyn.Resource(gvr),
			metricKilled: metricKilled,
			metricError:  metricErrors,
			resource:     resource,
		}
		registry.MustRegister(metricKilled, metricErrors)
		log.WithField("resource", resource.GetResourceName()).Infof("registered resource handler for %s/%s", resource.GetGroup(), resource.GetResourceName())
	}
	return handlers, nil
}
