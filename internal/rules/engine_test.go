package rules

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/marcotuna/adaptive-metrics/internal/config"
	"github.com/marcotuna/adaptive-metrics/internal/models"
)

func TestEngine_SaveAndGetRule(t *testing.T) {
	// Create a temporary directory for rules
	tempDir, err := ioutil.TempDir("", "rules-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config
	cfg := &config.Config{
		Aggregator: config.AggregatorConfig{
			RulesPath: tempDir,
		},
	}

	// Create engine
	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create a test rule
	rule := &models.Rule{
		ID:          "test-rule-1",
		Name:        "Test Rule",
		Description: "A test rule for unit testing",
		Enabled:     true,
		CreatedAt:   time.Now().Truncate(time.Second), // Truncate to avoid precision issues in comparison
		UpdatedAt:   time.Now().Truncate(time.Second),
		Matcher: models.MetricMatcher{
			MetricNames: []string{"http_requests_total"},
			Labels: map[string]string{
				"method": "GET",
			},
		},
		Aggregation: models.AggregationConfig{
			Type:            "sum",
			IntervalSeconds: 60,
			Segmentation:    []string{"path", "status_code"},
		},
		Output: models.OutputConfig{
			MetricName: "http_requests_aggregated",
			AdditionalLabels: map[string]string{
				"aggregated_by": "test",
			},
			DropOriginal: false,
		},
	}

	// Test SaveRule
	if err := engine.SaveRule(rule); err != nil {
		t.Fatalf("Failed to save rule: %v", err)
	}

	// Verify file was created
	ruleFilePath := filepath.Join(tempDir, rule.ID+".yaml")
	if _, err := os.Stat(ruleFilePath); os.IsNotExist(err) {
		t.Errorf("Rule file was not created at %s", ruleFilePath)
	}

	// Test GetRule
	retrievedRule, err := engine.GetRule(rule.ID)
	if err != nil {
		t.Fatalf("Failed to get rule: %v", err)
	}

	// Compare rules (simple comparison for essential fields)
	if retrievedRule.ID != rule.ID {
		t.Errorf("Retrieved rule ID = %v, want %v", retrievedRule.ID, rule.ID)
	}
	if retrievedRule.Name != rule.Name {
		t.Errorf("Retrieved rule Name = %v, want %v", retrievedRule.Name, rule.Name)
	}
	if retrievedRule.Description != rule.Description {
		t.Errorf("Retrieved rule Description = %v, want %v", retrievedRule.Description, rule.Description)
	}
	if retrievedRule.Enabled != rule.Enabled {
		t.Errorf("Retrieved rule Enabled = %v, want %v", retrievedRule.Enabled, rule.Enabled)
	}
}

func TestEngine_UpdateRule(t *testing.T) {
	// Create a temporary directory for rules
	tempDir, err := ioutil.TempDir("", "rules-update-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config
	cfg := &config.Config{
		Aggregator: config.AggregatorConfig{
			RulesPath: tempDir,
		},
	}

	// Create engine
	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create a test rule
	originalRule := &models.Rule{
		ID:          "test-rule-update",
		Name:        "Original Rule",
		Description: "Original description",
		Enabled:     true,
		CreatedAt:   time.Now().Truncate(time.Second),
		UpdatedAt:   time.Now().Truncate(time.Second),
		Matcher: models.MetricMatcher{
			MetricNames: []string{"original_metric"},
		},
		Aggregation: models.AggregationConfig{
			Type:            "sum",
			IntervalSeconds: 60,
		},
		Output: models.OutputConfig{
			MetricName: "original_aggregated",
		},
	}

	// Save the original rule
	if err := engine.SaveRule(originalRule); err != nil {
		t.Fatalf("Failed to save original rule: %v", err)
	}

	// Create an updated version of the rule
	updatedRule := &models.Rule{
		ID:          "test-rule-update", // Same ID
		Name:        "Updated Rule",
		Description: "Updated description",
		Enabled:     false,
		CreatedAt:   originalRule.CreatedAt,
		UpdatedAt:   time.Now().Truncate(time.Second),
		Matcher: models.MetricMatcher{
			MetricNames: []string{"updated_metric"},
		},
		Aggregation: models.AggregationConfig{
			Type:            "avg",
			IntervalSeconds: 120,
		},
		Output: models.OutputConfig{
			MetricName: "updated_aggregated",
		},
	}

	// Update the rule
	if err := engine.UpdateRule(updatedRule); err != nil {
		t.Fatalf("Failed to update rule: %v", err)
	}

	// Retrieve the updated rule
	retrievedRule, err := engine.GetRule(updatedRule.ID)
	if err != nil {
		t.Fatalf("Failed to get updated rule: %v", err)
	}

	// Verify updates
	if retrievedRule.Name != updatedRule.Name {
		t.Errorf("Retrieved rule Name = %v, want %v", retrievedRule.Name, updatedRule.Name)
	}
	if retrievedRule.Description != updatedRule.Description {
		t.Errorf("Retrieved rule Description = %v, want %v", retrievedRule.Description, updatedRule.Description)
	}
	if retrievedRule.Enabled != updatedRule.Enabled {
		t.Errorf("Retrieved rule Enabled = %v, want %v", retrievedRule.Enabled, updatedRule.Enabled)
	}
	if retrievedRule.Matcher.MetricNames[0] != updatedRule.Matcher.MetricNames[0] {
		t.Errorf("Retrieved rule MetricNames[0] = %v, want %v", retrievedRule.Matcher.MetricNames[0], updatedRule.Matcher.MetricNames[0])
	}
	if retrievedRule.Aggregation.Type != updatedRule.Aggregation.Type {
		t.Errorf("Retrieved rule Aggregation.Type = %v, want %v", retrievedRule.Aggregation.Type, updatedRule.Aggregation.Type)
	}
	if retrievedRule.Output.MetricName != updatedRule.Output.MetricName {
		t.Errorf("Retrieved rule Output.MetricName = %v, want %v", retrievedRule.Output.MetricName, updatedRule.Output.MetricName)
	}
}

func TestEngine_DeleteRule(t *testing.T) {
	// Create a temporary directory for rules
	tempDir, err := ioutil.TempDir("", "rules-delete-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config
	cfg := &config.Config{
		Aggregator: config.AggregatorConfig{
			RulesPath: tempDir,
		},
	}

	// Create engine
	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create a test rule
	rule := &models.Rule{
		ID:          "test-rule-delete",
		Name:        "Rule to Delete",
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Matcher: models.MetricMatcher{
			MetricNames: []string{"metric_to_delete"},
		},
		Aggregation: models.AggregationConfig{
			Type:            "sum",
			IntervalSeconds: 60,
		},
		Output: models.OutputConfig{
			MetricName: "delete_aggregated",
		},
	}

	// Save the rule
	if err := engine.SaveRule(rule); err != nil {
		t.Fatalf("Failed to save rule: %v", err)
	}

	// Verify the rule exists
	if _, err := engine.GetRule(rule.ID); err != nil {
		t.Fatalf("Failed to get rule before deletion: %v", err)
	}

	// Delete the rule
	if err := engine.DeleteRule(rule.ID); err != nil {
		t.Fatalf("Failed to delete rule: %v", err)
	}

	// Verify the rule no longer exists
	if _, err := engine.GetRule(rule.ID); err == nil {
		t.Errorf("Expected error when getting deleted rule, but got nil")
	}

	// Verify the rule file is deleted
	ruleFilePath := filepath.Join(tempDir, rule.ID+".yaml")
	if _, err := os.Stat(ruleFilePath); !os.IsNotExist(err) {
		t.Errorf("Rule file still exists after deletion at %s", ruleFilePath)
	}
}

func TestEngine_GetRules(t *testing.T) {
	// Create a temporary directory for rules
	tempDir, err := ioutil.TempDir("", "rules-list-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test config
	cfg := &config.Config{
		Aggregator: config.AggregatorConfig{
			RulesPath: tempDir,
		},
	}

	// Create engine
	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create test rules
	rule1 := &models.Rule{
		ID:      "test-rule-1",
		Name:    "Rule 1",
		Enabled: true,
		Matcher: models.MetricMatcher{
			MetricNames: []string{"metric1"},
		},
		Aggregation: models.AggregationConfig{
			Type:            "sum",
			IntervalSeconds: 60,
		},
		Output: models.OutputConfig{
			MetricName: "metric1_aggregated",
		},
	}

	rule2 := &models.Rule{
		ID:      "test-rule-2",
		Name:    "Rule 2",
		Enabled: false,
		Matcher: models.MetricMatcher{
			MetricNames: []string{"metric2"},
		},
		Aggregation: models.AggregationConfig{
			Type:            "avg",
			IntervalSeconds: 120,
		},
		Output: models.OutputConfig{
			MetricName: "metric2_aggregated",
		},
	}

	// Save the rules
	if err := engine.SaveRule(rule1); err != nil {
		t.Fatalf("Failed to save rule1: %v", err)
	}
	if err := engine.SaveRule(rule2); err != nil {
		t.Fatalf("Failed to save rule2: %v", err)
	}

	// Get all rules
	rules, err := engine.GetRules()
	if err != nil {
		t.Fatalf("Failed to get rules: %v", err)
	}

	// Verify we have the correct number of rules
	if len(rules) != 2 {
		t.Errorf("GetRules() returned %d rules, want 2", len(rules))
	}

	// Create a map for easy lookup by ID
	rulesMap := make(map[string]*models.Rule)
	for _, rule := range rules {
		rulesMap[rule.ID] = rule
	}

	// Verify rule1 exists in the result
	if r, exists := rulesMap[rule1.ID]; !exists {
		t.Errorf("Rule with ID %s not found in GetRules() result", rule1.ID)
	} else if r.Name != rule1.Name {
		t.Errorf("Rule1 name = %v, want %v", r.Name, rule1.Name)
	}

	// Verify rule2 exists in the result
	if r, exists := rulesMap[rule2.ID]; !exists {
		t.Errorf("Rule with ID %s not found in GetRules() result", rule2.ID)
	} else if r.Name != rule2.Name {
		t.Errorf("Rule2 name = %v, want %v", r.Name, rule2.Name)
	}
}

// TestLoadRulesFromDisk tests that engine correctly loads rules from the rules directory
func TestLoadRulesFromDisk(t *testing.T) {
	// Create a temporary directory for rules
	tempDir, err := ioutil.TempDir("", "rules-load-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a rule file directly
	ruleContent := `
id: disk-rule-1
name: Disk Rule 1
description: Rule created on disk for testing
enabled: true
matcher:
  metric_names:
    - disk_metric
  labels:
    test: true
aggregation:
  type: sum
  interval_seconds: 60
  segmentation:
    - instance
output:
  metric_name: disk_metric_aggregated
  additional_labels:
    source: test
  drop_original: false
`
	ruleFilePath := filepath.Join(tempDir, "disk-rule-1.yaml")
	err = ioutil.WriteFile(ruleFilePath, []byte(ruleContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write rule file: %v", err)
	}

	// Create another non-YAML file to test that it's ignored
	nonYamlFilePath := filepath.Join(tempDir, "not-a-rule.txt")
	err = ioutil.WriteFile(nonYamlFilePath, []byte("not a rule"), 0644)
	if err != nil {
		t.Fatalf("Failed to write non-YAML file: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		Aggregator: config.AggregatorConfig{
			RulesPath: tempDir,
		},
	}

	// Create engine which should load the rules
	engine, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Verify the rule was loaded
	rule, err := engine.GetRule("disk-rule-1")
	if err != nil {
		t.Fatalf("Failed to get loaded rule: %v", err)
	}

	// Verify rule properties
	if rule.Name != "Disk Rule 1" {
		t.Errorf("Loaded rule name = %v, want %v", rule.Name, "Disk Rule 1")
	}
	if rule.Description != "Rule created on disk for testing" {
		t.Errorf("Loaded rule description = %v, want %v", rule.Description, "Rule created on disk for testing")
	}
	if !rule.Enabled {
		t.Errorf("Loaded rule enabled = %v, want %v", rule.Enabled, true)
	}
	if rule.Matcher.MetricNames[0] != "disk_metric" {
		t.Errorf("Loaded rule metric name = %v, want %v", rule.Matcher.MetricNames[0], "disk_metric")
	}
	if rule.Output.MetricName != "disk_metric_aggregated" {
		t.Errorf("Loaded rule output metric name = %v, want %v", rule.Output.MetricName, "disk_metric_aggregated")
	}
}