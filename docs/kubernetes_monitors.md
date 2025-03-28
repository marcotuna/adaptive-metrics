# Using Kubernetes Monitors with Adaptive Metrics

Adaptive Metrics can now integrate directly with your existing Kubernetes monitoring setup by modifying your `ServiceMonitor` and `PodMonitor` resources. This enables you to apply our intelligent metric aggregation recommendations while continuing to use your existing Prometheus Operator monitoring configuration.

## How It Works

This feature allows you to:

1. Generate new PodMonitor/ServiceMonitor resources that include relabeling rules for your aggregated metrics
2. Modify your existing monitoring resources to include the necessary metric relabeling rules
3. Optionally drop the original high-cardinality metrics once they're replaced by the aggregated version

## Modes of Operation

### Mode: Create

This mode generates a brand new PodMonitor or ServiceMonitor resource that includes relabeling rules for your aggregated metrics.

```yaml
output_kubernetes:
  enabled: true
  resource_type: "ServiceMonitor"
  mode: "create"
  namespace: "monitoring"
  labels:
    release: "prometheus"
  selector:
    app: "my-service"
  port: "metrics"
  path: "/metrics"
```

### Mode: Modify

This mode generates relabeling configurations that you can add to your existing PodMonitor or ServiceMonitor resources. It provides a template for updating your monitors to include the aggregated metrics and optionally drop the original high-cardinality ones.

```yaml
output_kubernetes:
  enabled: true
  resource_type: "ServiceMonitor"
  mode: "modify"
  namespace: "monitoring"
  existing_monitor_name: "my-service-monitor"
  drop_original_metrics: true
  original_metric_names:
    - "http_requests_total"
```

## Example Workflow

### 1. Identify High-Cardinality Metrics

First, analyze your metrics to identify high-cardinality metrics that would benefit from aggregation:

```bash
curl -X POST http://localhost:8080/api/v1/recommendations/generate
curl -X GET http://localhost:8080/api/v1/recommendations
```

### 2. Apply the Recommendation

When you find a recommendation you want to apply:

```bash
curl -X POST http://localhost:8080/api/v1/recommendations/{id}/apply
```

### 3. Generate the Monitor Configuration

After applying the recommendation, generate the Kubernetes monitor configuration:

```bash
# Get the rule ID first
curl -X GET http://localhost:8080/api/v1/rules

# Then generate the monitor configuration
curl -X GET http://localhost:8080/api/v1/rules/{rule-id}/kubernetes-monitor
```

### 4. Apply the Monitor Configuration

For new monitors:
```bash
kubectl apply -f servicemonitor-{rule-id}.yaml
```

For existing monitors, you'll need to merge the generated relabeling configurations into your existing monitor resources.

## Advanced Configuration

### Advanced Metric Relabeling

You can specify custom metric relabeling configurations:

```yaml
output_kubernetes:
  enabled: true
  resource_type: "ServiceMonitor"
  mode: "modify"
  existing_monitor_name: "my-service-monitor"
  metric_relabeling:
    - sourceLabels: ["__name__"]
      regex: "http_requests_aggregated"
      action: "keep"
    - sourceLabels: ["__name__"]
      regex: "http_requests_total"
      action: "drop"
```

### Connecting to Recommendation Engine

For a fully automated workflow, you can set the Kubernetes output configuration when applying a recommendation:

```bash
curl -X POST http://localhost:8080/api/v1/recommendations/{id}/apply -d '{
  "output_kubernetes": {
    "enabled": true,
    "resource_type": "ServiceMonitor",
    "mode": "modify",
    "existing_monitor_name": "my-service-monitor",
    "namespace": "monitoring",
    "drop_original_metrics": true
  }
}'
```

## Best Practices

1. **Start with modification mode** - Begin by modifying your existing monitors rather than creating new ones to maintain consistency with your current setup.

2. **Test before dropping originals** - Keep both the original and aggregated metrics until you've verified the aggregation is working correctly.

3. **Include team labels** - Add team or service labels to help track which aggregation rules are applied to each service.

4. **Monitor disk usage** - Track storage usage before and after applying aggregation to measure the actual impact.

5. **Document aggregation decisions** - Include comments in your monitor configurations explaining why certain metrics are aggregated.