package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/marcotuna/adaptive-metrics/internal/config"
	"github.com/marcotuna/adaptive-metrics/internal/models"
)

// Client represents a client for the Grafana Adaptive Metrics plugin
type Client struct {
	cfg        *config.PluginConfig
	httpClient *http.Client
}

// NewClient creates a new Grafana plugin client
func NewClient(cfg *config.PluginConfig) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// IsEnabled returns whether the plugin integration is enabled
func (c *Client) IsEnabled() bool {
	return c.cfg.Enabled
}

// GetStatus checks the plugin connection status
func (c *Client) GetStatus() (map[string]interface{}, error) {
	if !c.cfg.Enabled {
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/status", c.cfg.APIURL), nil)
	if err != nil {
		return nil, err
	}

	if c.cfg.AuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.cfg.AuthToken))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return map[string]interface{}{
			"enabled":  true,
			"connected": false,
			"error":    err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return map[string]interface{}{
			"enabled":  true,
			"connected": false,
			"error":    "Failed to decode response",
		}, nil
	}

	result["enabled"] = true
	result["connected"] = resp.StatusCode == http.StatusOK

	return result, nil
}

// SyncRules synchronizes rules with the Grafana plugin
func (c *Client) SyncRules(rules []*models.Rule) error {
	if !c.cfg.Enabled {
		return nil
	}

	data, err := json.Marshal(map[string]interface{}{
		"rules": rules,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/rules/sync", c.cfg.APIURL), bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.cfg.AuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.cfg.AuthToken))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to sync rules, status code: %d", resp.StatusCode)
	}

	return nil
}

// GetRecommendations fetches recommendations from the Grafana plugin
func (c *Client) GetRecommendations() ([]models.Recommendation, error) {
	if !c.cfg.Enabled {
		return []models.Recommendation{}, nil
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/recommendations", c.cfg.APIURL), nil)
	if err != nil {
		return nil, err
	}

	if c.cfg.AuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.cfg.AuthToken))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get recommendations, status code: %d", resp.StatusCode)
	}

	var result struct {
		Recommendations []models.Recommendation `json:"recommendations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Recommendations, nil
}