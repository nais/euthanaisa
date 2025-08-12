package metrics

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
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

func Register(pushGatewayURL string, registry *prometheus.Registry) *push.Pusher {
	registry.MustRegister(
		ResourcesScannedTotal,
		ResourceDeleteDuration,
		ResourcesKillableTotal,
		ResourceKilled,
		ResourceErrors,
	)

	if pushGatewayURL == "" {
		return nil
	}

	pusher := push.New(pushGatewayURL, "euthanaisa").
		Gatherer(registry).
		Grouping("namespace", os.Getenv("NAMESPACE")).
		Grouping("job", os.Getenv("CRONJOB_NAME"))

	return pusher
}
