package metrics

import (
	"github.com/nais/euthanaisa/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/sirupsen/logrus"
)

var (
	ResourcesScannedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "euthanaisa_resources_scanned_total",
			Help: "Total number of Kubernetes resources scanned by kind",
		},
		[]string{"resource"},
	)

	ResourceDeleteDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "euthanaisa_resource_delete_duration_seconds",
			Help:    "Histogram of time taken to delete a resource",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"resource"},
	)

	ResourcesKillableTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "euthanaisa_resources_killable_total",
			Help: "Total number of resources that are killable by euthanaisa",
		},
		[]string{"resource", "team"},
	)

	ResourceKilled = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "euthanaisa_killed_total",
			Help: "Number of Kubernetes resources killed by euthanaisa",
		},
		[]string{"group", "resource"},
	)

	ResourceErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "euthanaisa_errors_total",
			Help: "Number of errors encountered while processing resources in euthanaisa",
		},
		[]string{"group", "resource"},
	)
)

func Register(cfg config.MetricConfig, registry *prometheus.Registry, log *logrus.Entry) *push.Pusher {
	registry.MustRegister(
		ResourcesScannedTotal,
		ResourceDeleteDuration,
		ResourcesKillableTotal,
		ResourceKilled,
		ResourceErrors,
	)

	if !cfg.PushgatewayEnabled {
		return nil
	}

	pusher := push.New(cfg.PushgatewayEndpoint, "euthanaisa").Gatherer(registry)
	log.Infof("prometheus Pushgateway enabled, pushing metrics to %s", cfg.PushgatewayEndpoint)
	return pusher
}
