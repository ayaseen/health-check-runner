package monitoring

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all monitoring-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add monitoring-related checks here
	// In a real implementation, there would be actual monitoring checks
	// For example:
	// checks = append(checks, NewPrometheusCheck())
	// checks = append(checks, NewAlertManagerCheck())
	// checks = append(checks, NewGrafanaCheck())

	return checks
}
