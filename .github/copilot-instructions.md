<!-- Use this file to provide workspace-specific custom instructions to Copilot. For more details, visit https://code.visualstudio.com/docs/copilot/copilot-customization#_use-a-githubcopilotinstructionsmd-file -->

# Adaptive Metrics Project Guidelines

This project is a Go implementation of Grafana's Adaptive Metrics system for metric aggregation and cardinality reduction. When working in this codebase, please follow these guidelines:

## Code Organization
- Keep main business logic in the `internal/` directory
- Public APIs and utilities belong in the `pkg/` directory
- Command-line interfaces and service entry points go in `cmd/`
- Configuration files should be in `configs/`

## Coding Conventions
- Follow standard Go project layout and idioms
- Use meaningful variable and function names
- Include comments for exported functions and types
- Write tests for all new functionality
- Error handling should follow Go conventions (return errors, don't panic)

## Key Components
- **Rules Engine**: Manages metric aggregation rules
- **Metric Matcher**: Determines which rules apply to metrics
- **Aggregator**: Performs the actual metric aggregation
- **API Server**: Provides REST API for rule management
- **Storage**: Handles persistence of rules and metrics

## Metrics Handling
- Use Prometheus metrics format
- Properly track and expose internal metrics for monitoring
- Be mindful of performance implications when processing metrics

## Domain Model
The core domain model includes:
- **Rule**: Defines how to aggregate metrics
- **MetricSample**: Represents a single metric data point
- **AggregatedMetric**: Represents the result of aggregation

## Configuration
- Use structured configuration with sensible defaults
- Support environment variable overrides
- Keep sensitive information (like auth tokens) configurable

## Testing
- Write unit tests for core functionality
- Use benchmarks for performance-sensitive code
- Consider using integration tests for API endpoints