# Adaptive Metrics

A Go implementation of Grafana's Adaptive Metrics system for intelligent metric aggregation and cardinality reduction.

## Overview

Adaptive Metrics is designed to help reduce metric cardinality and storage costs in Prometheus/Grafana monitoring systems by intelligently aggregating high-cardinality metrics according to user-defined rules.

This project provides:

- A rule-based metrics aggregation system
- APIs for defining and managing aggregation rules
- Integration with Prometheus metrics format
- Support for diverse aggregation types (sum, avg, min, max, count)
- Customizable aggregation intervals and segmentation

## Features

- **Rule-Based Aggregation**: Define how metrics should be aggregated based on metric names and labels
- **Flexible Matching**: Match metrics using exact matching or regex patterns
- **Multiple Aggregation Types**: Support for sum, average, min, max, and count aggregations
- **Customizable Intervals**: Define aggregation intervals per rule
- **Label Segmentation**: Control how metrics are grouped during aggregation
- **Aggregation Delay**: Configure delay intervals to account for late-arriving samples
- **REST API**: API endpoints for rule management
- **Prometheus Integration**: Native integration with Prometheus metrics format
- **Monitoring**: Built-in metrics for monitoring the system itself

## Getting Started

### Prerequisites

- Go 1.21 or later

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/yourusername/adaptive-metrics.git
   cd adaptive-metrics
   ```

2. Build the application:
   ```
   go build -o adaptive-metrics
   ```

3. Run the server:
   ```
   ./adaptive-metrics
   ```

### Configuration

Adaptive Metrics uses a YAML configuration file. A default config will be created at `configs/config.yaml` on first run. You can override the config path using the `CONFIG_PATH` environment variable.

Example configuration:

```yaml
server:
  address: ":8080"
  read_timeout_seconds: 30
  write_timeout_seconds: 30

aggregator:
  batch_size: 1000
  aggregation_delay_ms: 60000  # 60 seconds
  worker_count: 5
  rules_path: "configs/rules"

storage:
  type: "memory"
  connection: ""

plugin:
  enabled: false
  api_url: "http://localhost:3000/api"
  auth_token: ""
```

## Creating Aggregation Rules

Rules can be defined via the API or as YAML files in the rules directory. Example rule:

```yaml
id: "example-rule"
name: "HTTP Requests Aggregation"
description: "Aggregate HTTP request metrics by status code"
enabled: true
matcher:
  metric_names:
    - "http_requests_total"
  labels:
    app: "my-app"
  label_regex:
    endpoint: "^/api/.*"
aggregation:
  type: "sum"
  interval_seconds: 60
  segmentation:
    - "status_code"
    - "method"
  delay_ms: 5000
output:
  metric_name: "http_requests_aggregated"
  additional_labels:
    aggregated_by: "adaptive_metrics"
  drop_original: false
```

## API Reference

The Adaptive Metrics API provides endpoints for managing aggregation rules:

- `GET /api/v1/rules`: List all rules
- `POST /api/v1/rules`: Create a new rule
- `GET /api/v1/rules/{id}`: Get a specific rule
- `PUT /api/v1/rules/{id}`: Update a rule
- `DELETE /api/v1/rules/{id}`: Delete a rule
- `GET /health`: Health check endpoint
- `GET /metrics`: Prometheus metrics endpoint

## License

[MIT License](LICENSE)