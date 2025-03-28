package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Aggregator  AggregatorConfig  `mapstructure:"aggregator"`
	Storage     StorageConfig     `mapstructure:"storage"`
	Plugin      PluginConfig      `mapstructure:"plugin"`
	RemoteWrite RemoteWriteConfig `mapstructure:"remote_write"`
	Logging     LoggingConfig     `mapstructure:"logging"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Address             string `mapstructure:"address"`
	Port                int    `mapstructure:"port"`
	ReadTimeoutSeconds  int    `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds int    `mapstructure:"write_timeout_seconds"`
	WebUIPath           string `mapstructure:"web_ui_path"`
}

// AggregatorConfig represents the metrics aggregation configuration
type AggregatorConfig struct {
	BatchSize          int    `mapstructure:"batch_size"`
	AggregationDelayMs int    `mapstructure:"aggregation_delay_ms"`
	WorkerCount        int    `mapstructure:"worker_count"`
	RulesPath          string `mapstructure:"rules_path"`
}

// StorageConfig represents the storage configuration
type StorageConfig struct {
	Type       string `mapstructure:"type"`
	Connection string `mapstructure:"connection"`
}

// PluginConfig represents the Grafana plugin configuration
type PluginConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	APIURL    string `mapstructure:"api_url"`
	AuthToken string `mapstructure:"auth_token"`
}

// RemoteWriteConfig represents the Prometheus remote write configuration
type RemoteWriteConfig struct {
	Enabled       bool              `mapstructure:"enabled"`
	Endpoints     []string          `mapstructure:"endpoints"`
	Username      string            `mapstructure:"username"`
	Password      string            `mapstructure:"password"`
	Headers       map[string]string `mapstructure:"headers"`
	MaxRetries    int               `mapstructure:"max_retries"`
	RetryInterval int               `mapstructure:"retry_interval_seconds"`
	BatchSize     int               `mapstructure:"batch_size"`
	Timeout       int               `mapstructure:"timeout_seconds"`
	// Controls whether to write only metrics from recommendations or all metrics
	RecommendationMetricsOnly bool `mapstructure:"recommendation_metrics_only"`
}

// LoggingConfig represents the logging configuration
type LoggingConfig struct {
	// Format determines the log output format: "json" or "text"
	Format string `mapstructure:"format"`
	// Level sets the minimum log level: "debug", "info", "warn", "error"
	Level string `mapstructure:"level"`
	// IncludeTimestamp controls whether timestamps are included in logs
	IncludeTimestamp bool `mapstructure:"include_timestamp"`
	// IncludeCaller controls whether caller information is included in logs
	IncludeCaller bool `mapstructure:"include_caller"`
	// File is the path to a log file (optional - logs to stdout if not specified)
	File string `mapstructure:"file"`
}

// Load loads the configuration from file and environment variables
func Load(customConfigPath string) (*Config, error) {
	// Set default config path
	configPath := "configs"
	if customConfigPath != "" {
		configPath = customConfigPath
	} else if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	// Config file name
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)
	viper.AddConfigPath(".")

	// Set defaults
	setDefaults()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, using defaults and env vars
	}

	// Environment variables can override config file
	viper.SetEnvPrefix("AM") // AM for Adaptive Metrics
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Create config directory if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// If config file doesn't exist, create a default one
	if _, err := os.Stat(filepath.Join(configPath, "config.yaml")); os.IsNotExist(err) {
		if err := viper.SafeWriteConfigAs(filepath.Join(configPath, "config.yaml")); err != nil {
			return nil, fmt.Errorf("failed to write default config file: %w", err)
		}
	}

	// Parse config into struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// setDefaults sets the default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.address", ":8080")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout_seconds", 30)
	viper.SetDefault("server.write_timeout_seconds", 30)
	viper.SetDefault("server.web_ui_path", "web/build")

	// Aggregator defaults
	viper.SetDefault("aggregator.batch_size", 1000)
	viper.SetDefault("aggregator.aggregation_delay_ms", 60000) // 60 seconds
	viper.SetDefault("aggregator.worker_count", 5)
	viper.SetDefault("aggregator.rules_path", "configs/rules")

	// Storage defaults
	viper.SetDefault("storage.type", "memory")
	viper.SetDefault("storage.connection", "")

	// Plugin defaults
	viper.SetDefault("plugin.enabled", false)
	viper.SetDefault("plugin.api_url", "http://localhost:3000/api")
	viper.SetDefault("plugin.auth_token", "")

	// Remote Write defaults
	viper.SetDefault("remote_write.enabled", false)
	viper.SetDefault("remote_write.endpoints", []string{})
	viper.SetDefault("remote_write.username", "")
	viper.SetDefault("remote_write.password", "")
	viper.SetDefault("remote_write.headers", map[string]string{})
	viper.SetDefault("remote_write.max_retries", 3)
	viper.SetDefault("remote_write.retry_interval_seconds", 30)
	viper.SetDefault("remote_write.batch_size", 1000)
	viper.SetDefault("remote_write.timeout_seconds", 30)
	viper.SetDefault("remote_write.recommendation_metrics_only", true)

	// Logging defaults
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.include_timestamp", true)
	viper.SetDefault("logging.include_caller", false)
	viper.SetDefault("logging.file", "")
}
