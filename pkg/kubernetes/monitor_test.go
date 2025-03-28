package kubernetes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marcotuna/adaptive-metrics/internal/models"
)

func TestGenerator_GenerateNewMonitor(t *testing.T) {
	// Create a temporary directory for test output
	tempDir, err := os.MkdirTemp("", "k8s-monitor-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create generator
	generator, err := NewGenerator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Create a test rule with Kubernetes output config
	rule := &models.Rule{
		ID:          "test-rule",
		Name:        "Test Rule",
		Description: "Test rule for Kubernetes monitor generation",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Matcher: models.MetricMatcher{
			MetricNames: []string{"http_requests_total"},
		},
		Aggregation: models.AggregationConfig{
			Type:            "sum",
			IntervalSeconds: 60,
			Segmentation:    []string{"status_code", "method"},
		},
		Output: models.OutputConfig{
			MetricName: "http_requests_aggregated",
			AdditionalLabels: map[string]string{
				"aggregated_by": "adaptive-metrics",
			},
		},
		OutputKubernetes: &models.KubernetesOutputConfig{
			Enabled:      true,
			ResourceType: "ServiceMonitor",
			Mode:         "create",
			Namespace:    "monitoring",
			Labels: map[string]string{
				"app": "adaptive-metrics",
			},
			Selector: map[string]string{
				"app": "my-app",
			},
			Port:      "metrics",
			Path:      "/metrics",
			Interval:  "30s",
			TLSConfig: nil,
			DropOriginalMetrics: true,
			OriginalMetricNames: []string{"http_requests_total"},
		},
	}

	// Generate the monitor
	filePath, err := generator.Generate(rule)
	if err != nil {
		t.Fatalf("Failed to generate monitor: %v", err)
	}

	// Check that the file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Monitor file not created at: %s", filePath)
	}

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read monitor file: %v", err)
	}

	// Verify the content contains expected elements
	contentStr := string(content)
	expectedElements := []string{
		"kind: ServiceMonitor",
		"metadata:",
		"name: http_requests_aggregated-monitor",
		"namespace: monitoring",
		"app: adaptive-metrics",
		"spec:",
		"selector:",
		"matchLabels:",
		"app: my-app",
		"endpoints:",
		"port: metrics",
		"path: /metrics",
		"interval: 30s",
		"metricRelabelings:",
		"__name__",
		"http_requests_aggregated",
		"action: keep",
		"http_requests_total",
		"action: drop",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Expected '%s' in generated monitor but it was not found", expected)
		}
	}
}

func TestGenerator_GenerateModifyMonitor(t *testing.T) {
	// Create a temporary directory for test output
	tempDir, err := os.MkdirTemp("", "k8s-monitor-modify-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create generator
	generator, err := NewGenerator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Create a test rule with Kubernetes output config in modify mode
	rule := &models.Rule{
		ID:          "test-rule-modify",
		Name:        "Test Rule Modify",
		Description: "Test rule for Kubernetes monitor modification",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Matcher: models.MetricMatcher{
			MetricNames: []string{"http_requests_total"},
		},
		Aggregation: models.AggregationConfig{
			Type:            "sum",
			IntervalSeconds: 60,
			Segmentation:    []string{"status_code", "method"},
		},
		Output: models.OutputConfig{
			MetricName: "http_requests_aggregated",
			AdditionalLabels: map[string]string{
				"aggregated_by": "adaptive-metrics",
			},
		},
		OutputKubernetes: &models.KubernetesOutputConfig{
			Enabled:      true,
			ResourceType: "PodMonitor",
			Mode:         "modify",
			Namespace:    "monitoring",
			ExistingMonitorName: "existing-pod-monitor",
			Port:      "metrics",
			Path:      "/metrics",
			DropOriginalMetrics: true,
			OriginalMetricNames: []string{"http_requests_total"},
		},
	}

	// Generate the monitor modification
	filePath, err := generator.Generate(rule)
	if err != nil {
		t.Fatalf("Failed to generate monitor modification: %v", err)
	}

	// Check that the file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Monitor modification file not created at: %s", filePath)
	}

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read monitor modification file: %v", err)
	}

	// Verify the content contains expected elements
	contentStr := string(content)
	expectedElements := []string{
		"# Applied modifications to PodMonitor: existing-pod-monitor",
		"kind: PodMonitor",
		"metadata:",
		"name: existing-pod-monitor",
		"namespace: monitoring",
		"spec:",
		"podMetricsEndpoints:",
		"metricRelabelings:",
		"__name__",
		"http_requests_aggregated",
		"action: keep",
		"http_requests_total",
		"action: drop",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Expected '%s' in generated monitor modification but it was not found", expected)
		}
	}
}

func TestGenerator_BuildMetricRelabelings(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "k8s-relabeling-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	generator, err := NewGenerator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	tests := []struct {
		name               string
		rule               *models.Rule
		expectKeepMetric   string
		expectDropMetrics  []string
		expectCustomConfig bool
	}{
		{
			name: "predefined relabeling config",
			rule: &models.Rule{
				Output: models.OutputConfig{
					MetricName: "custom_metric",
				},
				OutputKubernetes: &models.KubernetesOutputConfig{
					MetricRelabeling: []models.RelabelConfig{
						{
							SourceLabels: []string{"__name__"},
							Regex:        "custom_.*",
							Action:       "keep",
						},
						{
							SourceLabels: []string{"__name__"},
							Regex:        "drop_.*",
							Action:       "drop",
						},
					},
				},
			},
			expectCustomConfig: true,
		},
		{
			name: "auto-generated relabeling - drop original",
			rule: &models.Rule{
				Matcher: models.MetricMatcher{
					MetricNames: []string{"original_metric"},
				},
				Output: models.OutputConfig{
					MetricName: "aggregated_metric",
				},
				OutputKubernetes: &models.KubernetesOutputConfig{
					DropOriginalMetrics: true,
					OriginalMetricNames: []string{"original_metric"},
				},
			},
			expectKeepMetric:  "aggregated_metric",
			expectDropMetrics: []string{"original_metric"},
		},
		{
			name: "auto-generated relabeling - drop wildcards",
			rule: &models.Rule{
				Matcher: models.MetricMatcher{
					MetricNames: []string{"metric_*", "other_*", "*"},
				},
				Output: models.OutputConfig{
					MetricName: "combined_metric",
				},
				OutputKubernetes: &models.KubernetesOutputConfig{
					DropOriginalMetrics: true,
					// No explicit original metrics, should use matcher
				},
			},
			expectKeepMetric:  "combined_metric",
			expectDropMetrics: []string{"metric_", "other_"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relabelings := generator.buildMetricRelabelings(tt.rule)

			if tt.expectCustomConfig {
				// For predefined config, check if JSON structure is preserved
				if !strings.Contains(relabelings, `"action":`) {
					t.Errorf("Expected predefined relabeling config to be preserved")
				}
			} else {
				// For auto-generated config, check for keep and drop actions
				if tt.expectKeepMetric != "" && !strings.Contains(relabelings, tt.expectKeepMetric) {
					t.Errorf("Expected keep action for %s but not found", tt.expectKeepMetric)
				}

				for _, dropMetric := range tt.expectDropMetrics {
					if !strings.Contains(relabelings, dropMetric) {
						t.Errorf("Expected drop action for %s but not found", dropMetric)
					}
				}
			}
		})
	}
}

func TestRenderMonitor(t *testing.T) {
	// Create a simple rule for rendering
	rule := &models.Rule{
		ID:       "render-test",
		Name:     "Render Test",
		Enabled:  true,
		Matcher: models.MetricMatcher{
			MetricNames: []string{"test_metric"},
		},
		Output: models.OutputConfig{
			MetricName: "test_aggregated",
		},
		OutputKubernetes: &models.KubernetesOutputConfig{
			Enabled:      true,
			ResourceType: "ServiceMonitor",
			Mode:         "create",
			Namespace:    "test",
			Selector: map[string]string{
				"app": "test",
			},
			Port: "metrics",
		},
	}

	// Test RenderMonitor function
	output, err := RenderMonitor(rule)
	if err != nil {
		t.Fatalf("RenderMonitor failed: %v", err)
	}

	// Verify output is non-empty and contains expected elements
	if len(output) == 0 {
		t.Errorf("RenderMonitor returned empty output")
	}

	expectedElements := []string{
		"kind: ServiceMonitor",
		"metadata:",
		"name: test_aggregated-monitor",
		"namespace: test",
		"spec:",
		"selector:",
		"matchLabels:",
		"app: test",
		"port: metrics",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected '%s' in rendered monitor but it was not found", expected)
		}
	}
}

func TestWriteMonitorFile(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "k8s-write-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple rule
	rule := &models.Rule{
		ID:      "write-test",
		Name:    "Write Test",
		Enabled: true,
		Matcher: models.MetricMatcher{
			MetricNames: []string{"write_metric"},
		},
		Output: models.OutputConfig{
			MetricName: "write_aggregated",
		},
		OutputKubernetes: &models.KubernetesOutputConfig{
			Enabled:      true,
			ResourceType: "ServiceMonitor",
			Mode:         "create",
			Namespace:    "write-test",
			Selector: map[string]string{
				"app": "write-test",
			},
			Port: "metrics",
		},
	}

	// Test WriteMonitorFile function
	filePath, err := WriteMonitorFile(rule, tempDir)
	if err != nil {
		t.Fatalf("WriteMonitorFile failed: %v", err)
	}

	// Verify file was created at the expected path
	expectedFilename := filepath.Join(tempDir, "ServiceMonitor-write-test.yaml")
	if filePath != expectedFilename {
		t.Errorf("WriteMonitorFile returned path %s, expected %s", filePath, expectedFilename)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Monitor file not found at expected path: %s", filePath)
	}

	// Verify file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read monitor file: %v", err)
	}

	contentStr := string(content)
	expectedElements := []string{
		"kind: ServiceMonitor",
		"metadata:",
		"name: write_aggregated-monitor",
		"namespace: write-test",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Expected '%s' in written monitor file but it was not found", expected)
		}
	}
}

func TestGeneratorErrors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "k8s-errors-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	generator, err := NewGenerator(tempDir)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Test with nil rule
	_, err = generator.Generate(nil)
	if err == nil {
		t.Error("Expected error for nil rule but got nil")
	}

	// Test with nil Kubernetes output config
	_, err = generator.Generate(&models.Rule{
		ID:      "error-test-1",
		Name:    "Error Test 1",
		Enabled: true,
		OutputKubernetes: nil,
	})
	if err == nil {
		t.Error("Expected error for nil OutputKubernetes but got nil")
	}

	// Test with disabled Kubernetes output
	_, err = generator.Generate(&models.Rule{
		ID:      "error-test-2",
		Name:    "Error Test 2",
		Enabled: true,
		OutputKubernetes: &models.KubernetesOutputConfig{
			Enabled: false,
		},
	})
	if err == nil {
		t.Error("Expected error for disabled OutputKubernetes but got nil")
	}

	// Test with invalid resource type
	_, err = generator.Generate(&models.Rule{
		ID:      "error-test-3",
		Name:    "Error Test 3",
		Enabled: true,
		OutputKubernetes: &models.KubernetesOutputConfig{
			Enabled:      true,
			ResourceType: "InvalidType", // Should be PodMonitor or ServiceMonitor
			Mode:         "create",
		},
	})
	if err == nil {
		t.Error("Expected error for invalid resource type but got nil")
	}

	// Test modify mode without existing monitor name
	_, err = generator.Generate(&models.Rule{
		ID:      "error-test-4",
		Name:    "Error Test 4",
		Enabled: true,
		OutputKubernetes: &models.KubernetesOutputConfig{
			Enabled:      true,
			ResourceType: "ServiceMonitor",
			Mode:         "modify",
			ExistingMonitorName: "", // Missing required field for modify mode
		},
	})
	if err == nil {
		t.Error("Expected error for missing existing monitor name in modify mode but got nil")
	}
}