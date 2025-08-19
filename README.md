# euthanaisa

Euthanaisa is a Kubernetes utility that loops through all defined resources in the cluster looking for the annotation
`euthanaisa.nais.io/kill-after: <timestamp>`.

If it finds this annotation, and the timestamp is valid and earlier than `time.Now()`, it deletes the resource.

If the resource has an owner reference to a `nais.io/Application`, it will delete the owner resource instead.

On completion, it pushes metrics to Prometheus Pushgateway, prefixed with `euthanaisa`.

## Features

- Scans Kubernetes resources for the `kill-after` annotation.
- Deletes resources based on the annotation timestamp.
- Handles owner references to ensure proper deletion of dependent resources.
- Pushes metrics to Prometheus Pushgateway for monitoring.

### Cluster Requirements

- **RBAC Permissions**: Ensure the tool has permissions to list, delete, and access resources in the cluster.
- **Annotations**: Resources must include the euthanaisa.nais.io/kill-after annotation for processing.

## Installation and Setup

### Prerequisites

- **Go**: Ensure Go is installed (minimum version required specified in `go.mod`).
- **Kubernetes Cluster**: The tool requires access to a Kubernetes cluster with appropriate RBAC permissions.
- **Prometheus Pushgateway**: A running instance of Prometheus Pushgateway to push metrics if enabled.

### Build, Run and Test

To build and run and push metrics locally:

```bash
  make local
```

To build a Linux binary:

```bash
  make linux-binary
```

To test the project, you can use the following command:

```bash
  make test
```

or for coverage:

```bash
  make test-coverage
```

### Configuration

The following flags can be configured when running the application:

- log-level: Set the logging level (e.g., debug, info, error).
- pushgateway-url: Specify the URL of the Prometheus Pushgateway.
- log-format: Set the log format (e.g., json, text).
- resources-file: Path to the resources configuration file (default: `/app/config/resources.yaml`).

#### Resources Configuration

The `resources.yaml` file defines the Kubernetes resources that the tool will process. Each resource entry specifies the
kind, API version, and other relevant details.

#### Example `resources.yaml`

```yaml
resources:
  - kind: Deployment
    resource: deployments
    group: apps
    apiVersion: v1
    ownedBy:
      - Application
```

### Metrics

The following metrics are pushed to Prometheus Pushgateway:

- **`euthanaisa_resources_scanned_total{resource}`**: Total number of Kubernetes resources scanned by kind.
- **`euthanaisa_resource_delete_duration_seconds{resource}`**: Histogram of time taken to delete a resource.
- **`euthanaisa_resources_killable_total{resource, namespace}`**: Total number of resources that are killable by euthanaisa.
- **`euthanaisa_killed_total{group, resource}`**: Number of Kubernetes resources killed by euthanaisa.
- **`euthanaisa_errors_total{group, resource}`**: Number of errors encountered while processing resources in euthanaisa.
