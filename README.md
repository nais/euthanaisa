# euthanaisa

Euthanaisa loops through all deployments in the cluster looking for the annotation `euthanaisa.nais.io/kill-after: <timestamp>`. 

If it finds this annotation, and the timestamp is valid and before `time.Now()`, it deletes the deployment.

If the deployment has a owner reference to a `nais.io/Application` it will delete this instead. 

On completion it will push metrics to Prometheus Pushgateway, prefixed with `euthanaisa`. 
