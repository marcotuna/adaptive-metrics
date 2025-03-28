package kubernetes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/marcotuna/adaptive-metrics/internal/models"
)

// Generator creates Kubernetes monitor resources for metrics
type Generator struct {
	outputDir string
}

// NewGenerator creates a new Kubernetes resource generator
func NewGenerator(outputDir string) (*Generator, error) {
	// Ensure the output directory exists
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	return &Generator{
		outputDir: outputDir,
	}, nil
}

// Generate creates or modifies a Kubernetes monitoring resource for a rule
func (g *Generator) Generate(rule *models.Rule) (string, error) {
	if rule == nil || rule.OutputKubernetes == nil || !rule.OutputKubernetes.Enabled {
		return "", fmt.Errorf("rule does not have Kubernetes output enabled")
	}

	config := rule.OutputKubernetes

	switch config.Mode {
	case "create":
		return g.generateNewMonitor(rule)
	case "modify", "patch":
		return g.modifyExistingMonitor(rule)
	default:
		// Default to create if mode not specified
		return g.generateNewMonitor(rule)
	}
}

// generateNewMonitor creates a new ServiceMonitor or PodMonitor
func (g *Generator) generateNewMonitor(rule *models.Rule) (string, error) {
	// Determine which template to use based on resource type
	var tmpl *template.Template
	var err error

	switch rule.OutputKubernetes.ResourceType {
	case "PodMonitor":
		tmpl, err = template.New("podmonitor").Parse(newPodMonitorTemplate)
	case "ServiceMonitor":
		tmpl, err = template.New("servicemonitor").Parse(newServiceMonitorTemplate)
	default:
		return "", fmt.Errorf("unsupported resource type: %s", rule.OutputKubernetes.ResourceType)
	}

	if err != nil {
		return "", fmt.Errorf("failed to create template: %w", err)
	}

	// Build metric relabelings
	metricRelabelings := g.buildMetricRelabelings(rule)

	// Prepare data for template
	data := map[string]interface{}{
		"Rule":              rule,
		"K8sConfig":         rule.OutputKubernetes,
		"MetricRelabelings": metricRelabelings,
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Write to file if output directory is specified
	if g.outputDir != "" {
		filename := filepath.Join(g.outputDir, fmt.Sprintf("%s-%s.yaml",
			rule.OutputKubernetes.ResourceType, rule.ID))

		if err := ioutil.WriteFile(filename, buf.Bytes(), 0644); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}

		return filename, nil
	}

	return buf.String(), nil
}

// modifyExistingMonitor reads an existing monitor and modifies it based on the rule
func (g *Generator) modifyExistingMonitor(rule *models.Rule) (string, error) {
	config := rule.OutputKubernetes

	// Check if the existing monitor name is provided
	if config.ExistingMonitorName == "" {
		return "", fmt.Errorf("existing monitor name must be provided for modify/patch mode")
	}

	// For this example, we'll use a template to show how to modify an existing monitor
	// In a real implementation, you would read the existing file, unmarshal it, modify it, and write it back
	var tmpl *template.Template
	var err error

	switch config.ResourceType {
	case "PodMonitor":
		tmpl, err = template.New("modify-podmonitor").Parse(modifyPodMonitorTemplate)
	case "ServiceMonitor":
		tmpl, err = template.New("modify-servicemonitor").Parse(modifyServiceMonitorTemplate)
	default:
		return "", fmt.Errorf("unsupported resource type: %s", config.ResourceType)
	}

	if err != nil {
		return "", fmt.Errorf("failed to create template: %w", err)
	}

	// Build metric relabelings
	metricRelabelings := g.buildMetricRelabelings(rule)

	// Prepare data for template
	data := map[string]interface{}{
		"Rule":              rule,
		"K8sConfig":         config,
		"MetricRelabelings": metricRelabelings,
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Write to file if output directory is specified
	if g.outputDir != "" {
		filename := filepath.Join(g.outputDir, fmt.Sprintf("modified-%s-%s.yaml",
			config.ResourceType, config.ExistingMonitorName))

		if err := ioutil.WriteFile(filename, buf.Bytes(), 0644); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}

		return filename, nil
	}

	return buf.String(), nil
}

// buildMetricRelabelings creates the appropriate metric relabeling configurations
func (g *Generator) buildMetricRelabelings(rule *models.Rule) string {
	var relabelings []string
	config := rule.OutputKubernetes

	// If there are predefined relabelings in the config, use them
	if len(config.MetricRelabeling) > 0 {
		relabelingsBytes, _ := json.MarshalIndent(config.MetricRelabeling, "", "  ")
		return string(relabelingsBytes)
	}

	// Add a relabeling to keep the aggregated metric
	keepAggregated := fmt.Sprintf(`
- sourceLabels: [__name__]
  regex: %s
  action: keep`, rule.Output.MetricName)
	relabelings = append(relabelings, keepAggregated)

	// If drop original metrics is enabled, add relabelings to drop them
	if config.DropOriginalMetrics && len(config.OriginalMetricNames) > 0 {
		for _, originalMetric := range config.OriginalMetricNames {
			dropOriginal := fmt.Sprintf(`
- sourceLabels: [__name__]
  regex: %s
  action: drop`, originalMetric)
			relabelings = append(relabelings, dropOriginal)
		}
	}

	// Default: if no original metrics specified but drop is enabled, try to drop metrics from matcher
	if config.DropOriginalMetrics && len(config.OriginalMetricNames) == 0 && len(rule.Matcher.MetricNames) > 0 {
		for _, metricName := range rule.Matcher.MetricNames {
			if metricName != "*" { // Skip wildcard matches
				dropOriginal := fmt.Sprintf(`
- sourceLabels: [__name__]
  regex: %s
  action: drop`, metricName)
				relabelings = append(relabelings, dropOriginal)
			}
		}
	}

	return fmt.Sprintf(`%s`, strings.Join(relabelings, "\n"))
}

// New ServiceMonitor template based on Prometheus Operator CRD
const newServiceMonitorTemplate = `apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: {{ .Rule.Output.MetricName }}-monitor
  namespace: {{ .K8sConfig.Namespace }}
  labels:
    {{- range $key, $value := .K8sConfig.Labels }}
    {{ $key }}: {{ $value }}
    {{- end }}
spec:
  selector:
    matchLabels:
      {{- range $key, $value := .K8sConfig.Selector }}
      {{ $key }}: {{ $value }}
      {{- end }}
  endpoints:
  - port: {{ .K8sConfig.Port }}
    {{- if .K8sConfig.Path }}
    path: {{ .K8sConfig.Path }}
    {{- end }}
    {{- if .K8sConfig.Interval }}
    interval: {{ .K8sConfig.Interval }}
    {{- end }}
    {{- if .K8sConfig.TLSConfig }}
    tlsConfig:
      {{- if .K8sConfig.TLSConfig.InsecureSkipVerify }}
      insecureSkipVerify: {{ .K8sConfig.TLSConfig.InsecureSkipVerify }}
      {{- end }}
      {{- if .K8sConfig.TLSConfig.CAFile }}
      caFile: {{ .K8sConfig.TLSConfig.CAFile }}
      {{- end }}
      {{- if .K8sConfig.TLSConfig.CertFile }}
      certFile: {{ .K8sConfig.TLSConfig.CertFile }}
      {{- end }}
      {{- if .K8sConfig.TLSConfig.KeyFile }}
      keyFile: {{ .K8sConfig.TLSConfig.KeyFile }}
      {{- end }}
      {{- if .K8sConfig.TLSConfig.ServerName }}
      serverName: {{ .K8sConfig.TLSConfig.ServerName }}
      {{- end }}
    {{- end }}
    metricRelabelings:
    {{ .MetricRelabelings }}
`

// New PodMonitor template based on Prometheus Operator CRD
const newPodMonitorTemplate = `apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: {{ .Rule.Output.MetricName }}-monitor
  namespace: {{ .K8sConfig.Namespace }}
  labels:
    {{- range $key, $value := .K8sConfig.Labels }}
    {{ $key }}: {{ $value }}
    {{- end }}
spec:
  selector:
    matchLabels:
      {{- range $key, $value := .K8sConfig.Selector }}
      {{ $key }}: {{ $value }}
      {{- end }}
  podMetricsEndpoints:
  - port: {{ .K8sConfig.Port }}
    {{- if .K8sConfig.Path }}
    path: {{ .K8sConfig.Path }}
    {{- end }}
    {{- if .K8sConfig.Interval }}
    interval: {{ .K8sConfig.Interval }}
    {{- end }}
    {{- if .K8sConfig.TLSConfig }}
    tlsConfig:
      {{- if .K8sConfig.TLSConfig.InsecureSkipVerify }}
      insecureSkipVerify: {{ .K8sConfig.TLSConfig.InsecureSkipVerify }}
      {{- end }}
      {{- if .K8sConfig.TLSConfig.CAFile }}
      caFile: {{ .K8sConfig.TLSConfig.CAFile }}
      {{- end }}
      {{- if .K8sConfig.TLSConfig.CertFile }}
      certFile: {{ .K8sConfig.TLSConfig.CertFile }}
      {{- end }}
      {{- if .K8sConfig.TLSConfig.KeyFile }}
      keyFile: {{ .K8sConfig.TLSConfig.KeyFile }}
      {{- end }}
      {{- if .K8sConfig.TLSConfig.ServerName }}
      serverName: {{ .K8sConfig.TLSConfig.ServerName }}
      {{- end }}
    {{- end }}
    metricRelabelings:
    {{ .MetricRelabelings }}
`

// Modify ServiceMonitor template - shows how to patch an existing monitor
const modifyServiceMonitorTemplate = `# Applied modifications to ServiceMonitor: {{ .K8sConfig.ExistingMonitorName }}
# This is a patch to be applied to the existing ServiceMonitor
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: {{ .K8sConfig.ExistingMonitorName }}
  namespace: {{ .K8sConfig.Namespace }}
spec:
  endpoints:
  # Add these metricRelabelings to the appropriate endpoint in your ServiceMonitor
  - metricRelabelings:
    {{ .MetricRelabelings }}
`

// Modify PodMonitor template - shows how to patch an existing monitor
const modifyPodMonitorTemplate = `# Applied modifications to PodMonitor: {{ .K8sConfig.ExistingMonitorName }}
# This is a patch to be applied to the existing PodMonitor
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: {{ .K8sConfig.ExistingMonitorName }}
  namespace: {{ .K8sConfig.Namespace }}
spec:
  podMetricsEndpoints:
  # Add these metricRelabelings to the appropriate endpoint in your PodMonitor
  - metricRelabelings:
    {{ .MetricRelabelings }}
`

// RenderMonitor renders a monitor template as a string
func RenderMonitor(rule *models.Rule) (string, error) {
	gen, err := NewGenerator("")
	if err != nil {
		return "", err
	}
	return gen.Generate(rule)
}

// WriteMonitorFile generates a monitor file for a rule
func WriteMonitorFile(rule *models.Rule, outputDir string) (string, error) {
	gen, err := NewGenerator(outputDir)
	if err != nil {
		return "", err
	}
	return gen.Generate(rule)
}
