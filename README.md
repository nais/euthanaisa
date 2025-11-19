# euthanaisa

Euthanaisa is a Kubernetes utility that scans labeled resources and deletes those whose euthanaisa.nais.io/kill-after
timestamp has passed. The timestamp is an RFC3339 value, either set directly or derived from a TTL parsed as a Go
time.Duration.

If the resource has an owner reference, Euthanaisa deletes the referenced owner resource instead.

Cleanup is only performed for resources that opt in via the label euthanaisa.nais.io/enabled=true. When enabled,
Euthanaisa also pushes metrics to Prometheus Pushgateway with the euthanaisa prefix.

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

### Configuration

The following flags can be configured when running the application:

- log-level: Set the logging level (e.g., debug, info, error).
- pushgateway-endpoint: URL of the Prometheus Pushgateway instance.
- pushgateway-enabled: Enable or disable pushing metrics to Prometheus Pushgateway.
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
```