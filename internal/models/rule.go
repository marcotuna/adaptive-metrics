package models

import (
	"fmt"
	"time"
)

// Rule represents a metrics aggregation rule that matches Grafana's Adaptive Metrics format
type Rule struct {
	ID               string           `json:"id" yaml:"id"`
	Name             string           `json:"name" yaml:"name"`
	Description      string           `json:"description" yaml:"description"`
	Enabled          bool             `json:"enabled" yaml:"enabled"`
	CreatedAt        time.Time        `json:"created_at" yaml:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at" yaml:"updated_at"`
	
	// Matching criteria for metrics
	Matcher          MetricMatcher    `json:"matcher" yaml:"matcher"`
	
	// Aggregation configuration
	Aggregation      AggregationConfig `json:"aggregation" yaml:"aggregation"`
	
	// Output configuration
	Output           OutputConfig     `json:"output" yaml:"output"`
	
	// Kubernetes output configuration (optional)
	OutputKubernetes *KubernetesOutputConfig `json:"output_kubernetes,omitempty" yaml:"output_kubernetes,omitempty"`
	
	// Recommendation metadata (for Grafana compatibility)
	RecommendationID string           `json:"recommendation_id,omitempty" yaml:"recommendation_id,omitempty"`
	Source           string           `json:"source,omitempty" yaml:"source,omitempty"`
	Confidence       float64          `json:"confidence,omitempty" yaml:"confidence,omitempty"`
	EstimatedImpact  *EstimatedImpact `json:"estimated_impact,omitempty" yaml:"estimated_impact,omitempty"`
}

// EstimatedImpact represents the estimated impact of applying a rule
type EstimatedImpact struct {
	CardinalityReduction float64 `json:"cardinality_reduction" yaml:"cardinality_reduction"`
	SavingsPercentage    float64 `json:"savings_percentage" yaml:"savings_percentage"`
	AffectedSeries       int     `json:"affected_series" yaml:"affected_series"`
	RetentionPeriod      string  `json:"retention_period,omitempty" yaml:"retention_period,omitempty"`
}

// MetricMatcher defines criteria for matching metrics to be aggregated
type MetricMatcher struct {
	MetricNames []string          `json:"metric_names" yaml:"metric_names"`
	Labels      map[string]string `json:"labels" yaml:"labels"`
	LabelRegex  map[string]string `json:"label_regex" yaml:"label_regex"`
	// Grafana-specific matcher options
	IncludeMetaLabels bool              `json:"include_meta_labels,omitempty" yaml:"include_meta_labels,omitempty"`
	ExcludeLabels     []string          `json:"exclude_labels,omitempty" yaml:"exclude_labels,omitempty"`
}

// AggregationConfig defines how metrics should be aggregated
type AggregationConfig struct {
	// Aggregation type: sum, avg, min, max, count
	Type string `json:"type" yaml:"type"`
	
	// The interval for aggregation in seconds
	IntervalSeconds int `json:"interval_seconds" yaml:"interval_seconds"`
	
	// Segmentation defines how to group metrics during aggregation
	Segmentation []string `json:"segmentation" yaml:"segmentation"`
	
	// Advanced segmentation settings (Grafana-specific)
	SegmentationLimit int               `json:"segmentation_limit,omitempty" yaml:"segmentation_limit,omitempty"`
	SegmentationRules []SegmentationRule `json:"segmentation_rules,omitempty" yaml:"segmentation_rules,omitempty"`
	
	// Delay in milliseconds before aggregation to account for late-arriving samples
	DelayMs int `json:"delay_ms" yaml:"delay_ms"`
}

// SegmentationRule defines advanced rules for segmenting metrics
type SegmentationRule struct {
	Label       string `json:"label" yaml:"label"`
	LimitType   string `json:"limit_type" yaml:"limit_type"` // "top", "bottom", "include", "exclude"
	Limit       int    `json:"limit,omitempty" yaml:"limit,omitempty"`
	Values      []string `json:"values,omitempty" yaml:"values,omitempty"`
}

// OutputConfig defines the output configuration for aggregated metrics
type OutputConfig struct {
	// The name of the aggregated metric
	MetricName string `json:"metric_name" yaml:"metric_name"`
	
	// Additional labels to add to the aggregated metric
	AdditionalLabels map[string]string `json:"additional_labels" yaml:"additional_labels"`
	
	// Whether to drop original metrics after aggregation
	DropOriginal bool `json:"drop_original" yaml:"drop_original"`
	
	// Grafana-specific output options
	KeepLabels []string `json:"keep_labels,omitempty" yaml:"keep_labels,omitempty"`
}

// KubernetesOutputConfig defines the configuration for generating Kubernetes monitoring resources
type KubernetesOutputConfig struct {
	// Whether to generate Kubernetes monitoring resources
	Enabled bool `json:"enabled" yaml:"enabled"`
	
	// The type of resource to generate or modify: "PodMonitor" or "ServiceMonitor"
	ResourceType string `json:"resource_type" yaml:"resource_type"`
	
	// Mode for handling existing resources: "create" (create new), "modify" (modify existing), or "patch" (apply changes)
	Mode string `json:"mode" yaml:"mode"`
	
	// Namespace where the monitor should be created or found
	Namespace string `json:"namespace" yaml:"namespace"`
	
	// Name of the existing monitor to modify (required for mode="modify" or "patch")
	ExistingMonitorName string `json:"existing_monitor_name,omitempty" yaml:"existing_monitor_name,omitempty"`
	
	// Labels to add to the monitor resource
	Labels map[string]string `json:"labels" yaml:"labels"`
	
	// Selector configuration for the monitor (only for new monitors)
	Selector map[string]string `json:"selector" yaml:"selector"`
	
	// Endpoint configuration
	Port string `json:"port" yaml:"port"`
	Path string `json:"path" yaml:"path"`
	
	// Interval for scraping
	Interval string `json:"interval" yaml:"interval"`
	
	// Advanced metric relabeling configuration
	MetricRelabeling []RelabelConfig `json:"metric_relabeling,omitempty" yaml:"metric_relabeling,omitempty"`
	
	// Whether to drop the original metrics
	DropOriginalMetrics bool `json:"drop_original_metrics" yaml:"drop_original_metrics"`
	
	// Original metric names to be dropped (if DropOriginalMetrics is true)
	OriginalMetricNames []string `json:"original_metric_names,omitempty" yaml:"original_metric_names,omitempty"`
	
	// TLS configuration
	TLSConfig *TLSConfig `json:"tls_config,omitempty" yaml:"tls_config,omitempty"`
}

// RelabelConfig represents a metric relabeling configuration
type RelabelConfig struct {
	SourceLabels []string `json:"source_labels,omitempty" yaml:"source_labels,omitempty"`
	Separator    string   `json:"separator,omitempty" yaml:"separator,omitempty"`
	TargetLabel  string   `json:"target_label,omitempty" yaml:"target_label,omitempty"`
	Regex        string   `json:"regex,omitempty" yaml:"regex,omitempty"`
	Modulus      uint64   `json:"modulus,omitempty" yaml:"modulus,omitempty"`
	Replacement  string   `json:"replacement,omitempty" yaml:"replacement,omitempty"`
	Action       string   `json:"action" yaml:"action"`
}

// TLSConfig represents TLS configuration for the monitor
type TLSConfig struct {
	InsecureSkipVerify bool   `json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	CAFile             string `json:"ca_file,omitempty" yaml:"ca_file,omitempty"`
	CertFile           string `json:"cert_file,omitempty" yaml:"cert_file,omitempty"`
	KeyFile            string `json:"key_file,omitempty" yaml:"key_file,omitempty"`
	ServerName         string `json:"server_name,omitempty" yaml:"server_name,omitempty"`
}

// MetricSample represents a single metric sample
type MetricSample struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels"`
}

// AggregatedMetric represents an aggregated metric result
type AggregatedMetric struct {
	Name       string            `json:"name"`
	Value      float64           `json:"value"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	Labels     map[string]string `json:"labels"`
	SourceRule string            `json:"source_rule"`
	Count      int               `json:"count"` // Number of samples aggregated
}

// Validate checks if the rule configuration is valid
func (r *Rule) Validate() error {
	// Check required fields
	if r.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	
	if len(r.Matcher.MetricNames) == 0 {
		return fmt.Errorf("at least one metric name must be specified")
	}
	
	// Validate aggregation type
	validTypes := map[string]bool{
		"sum":   true,
		"avg":   true,
		"min":   true,
		"max":   true,
		"count": true,
	}
	if !validTypes[r.Aggregation.Type] {
		return fmt.Errorf("invalid aggregation type: %s", r.Aggregation.Type)
	}
	
	// Validate interval
	if r.Aggregation.IntervalSeconds <= 0 {
		return fmt.Errorf("aggregation interval must be greater than 0")
	}
	
	// Validate segmentation rules if present
	for _, segRule := range r.Aggregation.SegmentationRules {
		if segRule.Label == "" {
			return fmt.Errorf("segmentation rule label is required")
		}
		
		validLimitTypes := map[string]bool{
			"top":     true,
			"bottom":  true,
			"include": true,
			"exclude": true,
		}
		if !validLimitTypes[segRule.LimitType] {
			return fmt.Errorf("invalid segmentation limit type: %s", segRule.LimitType)
		}
		
		if (segRule.LimitType == "top" || segRule.LimitType == "bottom") && segRule.Limit <= 0 {
			return fmt.Errorf("segmentation limit must be greater than 0 for type %s", segRule.LimitType)
		}
		
		if (segRule.LimitType == "include" || segRule.LimitType == "exclude") && len(segRule.Values) == 0 {
			return fmt.Errorf("segmentation values must be specified for type %s", segRule.LimitType)
		}
	}
	
	// Validate output
	if r.Output.MetricName == "" {
		return fmt.Errorf("output metric name is required")
	}
	
	return nil
}

// Recommendation represents a suggested aggregation rule from the recommendation engine
type Recommendation struct {
	ID              string          `json:"id"`
	CreatedAt       time.Time       `json:"created_at"`
	Rule            Rule            `json:"rule"`
	Confidence      float64         `json:"confidence"`
	EstimatedImpact *EstimatedImpact `json:"estimated_impact"`
	Source          string          `json:"source"`
	Status          string          `json:"status"` // "pending", "applied", "rejected"
}