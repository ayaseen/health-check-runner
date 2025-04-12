package monitoring

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all monitoring-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add Prometheus check
	checks = append(checks, NewPrometheusCheck())

	// Add AlertManager check
	checks = append(checks, NewAlertManagerCheck())

	// Add Grafana check
	checks = append(checks, NewGrafanaCheck())

	return checks
}
