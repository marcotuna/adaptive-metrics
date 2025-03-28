package rules

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/marcotuna/adaptive-metrics/internal/config"
	"github.com/marcotuna/adaptive-metrics/internal/models"
	"gopkg.in/yaml.v3"
)

// Engine is responsible for managing and processing metric rules
type Engine struct {
	cfg     *config.Config
	rules   map[string]*models.Rule
	ruleMu  sync.RWMutex
	matcher *Matcher
}

// NewEngine creates a new rule engine
func NewEngine(cfg *config.Config) (*Engine, error) {
	engine := &Engine{
		cfg:   cfg,
		rules: make(map[string]*models.Rule),
	}
	
	// Initialize rule matcher
	engine.matcher = NewMatcher(engine)

	// Load rules from disk if path exists
	if err := engine.loadRulesFromDisk(); err != nil {
		return nil, fmt.Errorf("failed to load rules: %w", err)
	}

	return engine, nil
}

// loadRulesFromDisk loads rule definitions from disk
func (e *Engine) loadRulesFromDisk() error {
	rulesPath := e.cfg.Aggregator.RulesPath
	
	// Check if directory exists
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		// Create the directory if it doesn't exist
		if err := os.MkdirAll(rulesPath, 0755); err != nil {
			return fmt.Errorf("failed to create rules directory: %w", err)
		}
		return nil // No rules to load
	}

	files, err := ioutil.ReadDir(rulesPath)
	if err != nil {
		return fmt.Errorf("failed to read rules directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".yaml" && filepath.Ext(file.Name()) != ".yml" {
			continue
		}

		rulePath := filepath.Join(rulesPath, file.Name())
		ruleData, err := ioutil.ReadFile(rulePath)
		if err != nil {
			return fmt.Errorf("failed to read rule file %s: %w", file.Name(), err)
		}

		var rule models.Rule
		if err := yaml.Unmarshal(ruleData, &rule); err != nil {
			return fmt.Errorf("failed to parse rule file %s: %w", file.Name(), err)
		}

		// Generate ID if not present
		if rule.ID == "" {
			rule.ID = generateID()
		}

		// Add to rules map
		e.ruleMu.Lock()
		e.rules[rule.ID] = &rule
		e.ruleMu.Unlock()
	}

	return nil
}

// SaveRule saves a rule and persists it to disk
func (e *Engine) SaveRule(rule *models.Rule) error {
	// Generate ID if not present
	if rule.ID == "" {
		rule.ID = generateID()
	}

	// Validate rule
	if err := rule.Validate(); err != nil {
		return err
	}

	// Add to rules map
	e.ruleMu.Lock()
	e.rules[rule.ID] = rule
	e.ruleMu.Unlock()

	// Persist to disk
	return e.saveRuleToDisk(rule)
}

// UpdateRule updates an existing rule
func (e *Engine) UpdateRule(rule *models.Rule) error {
	// Check if rule exists
	e.ruleMu.RLock()
	_, exists := e.rules[rule.ID]
	e.ruleMu.RUnlock()
	
	if !exists {
		return fmt.Errorf("rule with ID %s does not exist", rule.ID)
	}

	// Validate rule
	if err := rule.Validate(); err != nil {
		return err
	}

	// Update in rules map
	e.ruleMu.Lock()
	e.rules[rule.ID] = rule
	e.ruleMu.Unlock()

	// Persist to disk
	return e.saveRuleToDisk(rule)
}

// DeleteRule removes a rule
func (e *Engine) DeleteRule(id string) error {
	// Check if rule exists
	e.ruleMu.RLock()
	rule, exists := e.rules[id]
	e.ruleMu.RUnlock()
	
	if !exists {
		return fmt.Errorf("rule with ID %s does not exist", id)
	}

	// Remove from rules map
	e.ruleMu.Lock()
	delete(e.rules, id)
	e.ruleMu.Unlock()

	// Remove from disk
	rulesPath := e.cfg.Aggregator.RulesPath
	rulePath := filepath.Join(rulesPath, fmt.Sprintf("%s.yaml", rule.ID))
	
	if err := os.Remove(rulePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete rule file: %w", err)
	}

	return nil
}

// GetRule retrieves a rule by ID
func (e *Engine) GetRule(id string) (*models.Rule, error) {
	e.ruleMu.RLock()
	defer e.ruleMu.RUnlock()
	
	rule, exists := e.rules[id]
	if !exists {
		return nil, fmt.Errorf("rule with ID %s does not exist", id)
	}
	
	return rule, nil
}

// GetRules returns all rules
func (e *Engine) GetRules() ([]*models.Rule, error) {
	e.ruleMu.RLock()
	defer e.ruleMu.RUnlock()
	
	rules := make([]*models.Rule, 0, len(e.rules))
	for _, rule := range e.rules {
		rules = append(rules, rule)
	}
	
	return rules, nil
}

// saveRuleToDisk persists a rule to disk
func (e *Engine) saveRuleToDisk(rule *models.Rule) error {
	rulesPath := e.cfg.Aggregator.RulesPath
	
	// Create the directory if it doesn't exist
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		if err := os.MkdirAll(rulesPath, 0755); err != nil {
			return fmt.Errorf("failed to create rules directory: %w", err)
		}
	}

	// Marshal rule to YAML
	ruleData, err := yaml.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to marshal rule: %w", err)
	}

	// Save to disk
	rulePath := filepath.Join(rulesPath, fmt.Sprintf("%s.yaml", rule.ID))
	if err := ioutil.WriteFile(rulePath, ruleData, 0644); err != nil {
		return fmt.Errorf("failed to write rule file: %w", err)
	}

	return nil
}

// generateID generates a unique ID for a rule
func generateID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate random ID: %v", err))
	}
	return hex.EncodeToString(bytes)
}