package api

import (
	"github.com/marcotuna/adaptive-metrics/internal/models"
	"github.com/marcotuna/adaptive-metrics/internal/rules"
)

// RuleEngineAdapter adapts the rules.Engine to the RuleStore interface
type RuleEngineAdapter struct {
	engine *rules.Engine
}

// NewRuleEngineAdapter creates a new adapter for the rules engine
func NewRuleEngineAdapter(engine *rules.Engine) *RuleEngineAdapter {
	return &RuleEngineAdapter{
		engine: engine,
	}
}

// AddRule implements RuleStore.AddRule
func (a *RuleEngineAdapter) AddRule(rule models.Rule) error {
	return a.engine.SaveRule(&rule)
}

// GetRule implements RuleStore.GetRule
func (a *RuleEngineAdapter) GetRule(id string) (models.Rule, error) {
	rulePtr, err := a.engine.GetRule(id)
	if err != nil {
		return models.Rule{}, err
	}
	return *rulePtr, nil
}

// GetRules implements RuleStore.GetRules
func (a *RuleEngineAdapter) GetRules() ([]models.Rule, error) {
	rulePtrs, err := a.engine.GetRules()
	if err != nil {
		return nil, err
	}

	rules := make([]models.Rule, len(rulePtrs))
	for i, rulePtr := range rulePtrs {
		rules[i] = *rulePtr
	}
	return rules, nil
}

// UpdateRule implements RuleStore.UpdateRule
func (a *RuleEngineAdapter) UpdateRule(rule models.Rule) error {
	return a.engine.UpdateRule(&rule)
}

// DeleteRule implements RuleStore.DeleteRule
func (a *RuleEngineAdapter) DeleteRule(id string) error {
	return a.engine.DeleteRule(id)
}
