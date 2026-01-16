# euthanaisa

Euthanaisa is a Kubernetes utility that deletes resources whose `euthanaisa.nais.io/kill-after` label timestamp has passed.

## Usage

Label any Kubernetes resource with a Unix timestamp (seconds since epoch):

```yaml
metadata:
  labels:
    euthanaisa.nais.io/kill-after: "1705320000"
```

Euthanaisa will delete the resource after the specified time.

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Log format (json, text) |
| `RESOURCES_FILE` | `/app/config/resources.yaml` | Path to resources config |
| `PUSHGATEWAY_ENDPOINT` | | Pushgateway URL (enables metrics push if set) |

### Resources Configuration

Define which Kubernetes resources to scan in `resources.yaml`:

```yaml
- resource: deployments
  group: apps
  version: v1
```
