package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var ResourcesScannedTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "resources_scanned_total",
		Help: "Total number of Kubernetes resources scanned by kind",
	},
	[]string{"resource"},
)

var ResourceDeleteDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "resource_delete_duration_seconds",
		Help:    "Histogram of time taken to delete a resource",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"resource"},
)

var ResourcesKillableTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "resources_killable_total",
		Help: "Total number of resources that were marked killable via kill-after annotation",
	},
	[]string{"resource", "namespace"},
)

func Register(pushGatewayURL string, registry *prometheus.Registry) *push.Pusher {
	registry.MustRegister(
		ResourcesScannedTotal,
		ResourceDeleteDuration,
	)

	var pusher *push.Pusher
	if pushGatewayURL != "" {
		pusher = push.New(pushGatewayURL, "euthanaisa").Gatherer(registry)
	}
	return pusher
}
