package euthanaiser

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type euthanaiser struct {
	appClient     dynamic.NamespaceableResourceInterface
	clientset     *kubernetes.Clientset
	log           *log.Entry
	appsKilled    prometheus.Counter
	deploysKilled prometheus.Counter
	errors        prometheus.Counter
	pusher        *push.Pusher
}

func New(logger *log.Entry, pushgatewayURL string) (*euthanaiser, error) {
	kc, err := kubeconfig()
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(kc)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}
	dyn, err := dynamic.NewForConfig(kc)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	registry := prometheus.NewRegistry()
	appsKilled := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "euthanaisa_apps_killed",
		Help: "Number of applications killed",
	})
	deploysKilled := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "euthanaisa_deploys_killed",
		Help: "Number of deployments killed",
	})
	errors := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "euthanaisa_errors",
		Help: "Number of errors",
	})

	registry.MustRegister(appsKilled, deploysKilled, errors)
	pusher := push.New(pushgatewayURL, "euthanaisa").Gatherer(registry)

	return &euthanaiser{
		log:           logger,
		clientset:     clientset,
		appClient:     dyn.Resource(schema.GroupVersionResource{Group: "nais.io", Version: "v1alpha1", Resource: "applications"}),
		pusher:        pusher,
		appsKilled:    appsKilled,
		deploysKilled: deploysKilled,
		errors:        errors,
	}, nil
}

func (e *euthanaiser) pushMetrics(ctx context.Context) {
	if err := e.pusher.AddContext(ctx); err != nil {
		e.log.WithError(err).Error("pushing metrics")
	}
	e.log.Info("pushed metrics to prometheus")
}

func (e *euthanaiser) Run(ctx context.Context) {
	defer e.pushMetrics(ctx)

	deploys, err := e.clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		e.log.WithError(err).Error("listing deployments")
		e.errors.Inc()
		return
	}
	for _, deploy := range deploys.Items {
		if err := e.process(ctx, deploy); err != nil {
			e.log.WithError(err).Error("processing deployment")
			e.errors.Inc()
		}
	}

	e.log.Info("finished processing all deployments")
}

func (e *euthanaiser) process(ctx context.Context, obj interface{}) error {
	deploy := obj.(appsv1.Deployment)
	if deploy.DeletionTimestamp != nil {
		return nil // Already being deleted
	}

	if deploy.Annotations["euthanaisa.nais.io/kill-after"] == "" {
		return nil
	}

	killAfter, err := time.Parse(time.RFC3339, deploy.Annotations["euthanaisa.nais.io/kill-after"])
	if err != nil {
		return err
	}

	if killAfter.After(time.Now()) {
		return nil
	}

	appOwnerRef := appOwnerRef(deploy)
	if appOwnerRef != nil { // We have a application owner reference, deleting this instead
		err := e.appClient.Namespace(deploy.Namespace).Delete(ctx, appOwnerRef.Name, metav1.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil // Already deleted
			}
			return fmt.Errorf("deleting application: %w", err)
		}
		e.appsKilled.Inc()
		log.Infof("deleted application %s in namespace: %s", appOwnerRef.Name, deploy.Namespace)
		return nil
	}

	if err := e.clientset.AppsV1().Deployments(deploy.Namespace).Delete(ctx, deploy.Name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("deleting deployment: %w", err)
	}
	e.deploysKilled.Inc()
	log.Debugf("deleted deployment %s in namespace %s", deploy.Name, deploy.Namespace)
	return nil
}

func kubeconfig() (*rest.Config, error) {
	if os.Getenv("KUBECONFIG") != "" {
		return clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	}

	return rest.InClusterConfig()
}

func appOwnerRef(deploy appsv1.Deployment) *metav1.OwnerReference {
	for _, ref := range deploy.OwnerReferences {
		if ref.Kind == "Application" {
			return &ref
		}
	}

	return nil
}
