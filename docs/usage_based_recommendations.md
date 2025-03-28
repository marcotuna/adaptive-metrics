# Usage-Based Rule Recommendations

Adaptive Metrics can now intelligently identify which metrics to aggregate based on actual usage patterns. This feature helps you optimize your metrics storage and processing by automatically suggesting rules for high-cardinality metrics.

## How It Works

1. **Usage Tracking**: The system tracks metric usage patterns, including:
   - Sample frequency
   - Cardinality (number of unique label combinations)
   - Label cardinality (number of unique values for each label)
   - Value ranges and patterns

2. **Analysis**: The recommendation engine analyzes this data to identify:
   - High-cardinality metrics that would benefit from aggregation
   - Optimal aggregation strategies based on metric behavior
   - Appropriate segmentation labels to maintain useful granularity

3. **Recommendations**: The system generates rule recommendations with:
   - Confidence scores
   - Estimated impact (cardinality reduction, storage savings)
   - Suggested aggregation parameters

## Using the Feature

### Generating Recommendations

Recommendations are generated automatically based on metric usage patterns. You can manually trigger recommendation generation via the API:

```bash
# Generate new recommendations based on current usage data
curl -X POST http://localhost:8080/api/v1/recommendations/generate
```

### Viewing Recommendations

```bash
# List all recommendations
curl -X GET http://localhost:8080/api/v1/recommendations

# View a specific recommendation
curl -X GET http://localhost:8080/api/v1/recommendations/{id}
```

### Applying a Recommendation

When you're ready to apply a recommendation, use the apply endpoint:

```bash
# Apply a specific recommendation
curl -X POST http://localhost:8080/api/v1/recommendations/{id}/apply
```

This will:
1. Create a new aggregation rule based on the recommendation
2. Mark the recommendation as "applied"
3. Enable the rule for immediate use

### Rejecting a Recommendation

If a recommendation isn't useful, you can reject it:

```bash
# Reject a specific recommendation
curl -X POST http://localhost:8080/api/v1/recommendations/{id}/reject
```

## Example Response

```json
{
  "recommendations": [
    {
      "id": "rec-a1b2c3d4",
      "created_at": "2025-03-26T14:30:00Z",
      "rule": {
        "id": "autogen-a1b2c3d4",
        "name": "Recommended aggregation for http_requests_total",
        "description": "Automatically generated rule to aggregate high-cardinality metric http_requests_total based on usage patterns",
        "enabled": false,
        "matcher": {
          "metric_names": ["http_requests_total"],
          "labels": {},
          "label_regex": {}
        },
        "aggregation": {
          "type": "sum",
          "interval_seconds": 60,
          "segmentation": ["status_code", "method"],
          "delay_ms": 5000
        },
        "output": {
          "metric_name": "http_requests_total_aggregated",
          "additional_labels": {
            "aggregated_by": "adaptive_metrics",
            "source": "usage_based_recommendation"
          },
          "drop_original": false
        },
        "source": "usage_analysis",
        "confidence": 0.85,
        "estimated_impact": {
          "cardinality_reduction": 12.5,
          "savings_percentage": 92.0,
          "affected_series": 500,
          "retention_period": "30d"
        }
      },
      "confidence": 0.85,
      "estimated_impact": {
        "cardinality_reduction": 12.5,
        "savings_percentage": 92.0,
        "affected_series": 500,
        "retention_period": "30d"
      },
      "source": "usage_analysis",
      "status": "pending"
    }
  ],
  "total": 1
}
```

## Configuration

The recommendation engine parameters can be adjusted in the application configuration:

```yaml
recommendation_engine:
  min_sample_threshold: 1000      # Minimum samples needed to generate a recommendation
  min_cardinality_threshold: 100  # Minimum cardinality needed to consider aggregation
  min_confidence: 0.5             # Minimum confidence score for a recommendation
  usage_retention_period: "90d"   # How long to retain usage data
```

These thresholds help ensure that recommendations are based on sufficient data and will have meaningful impact.