package monitoring

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all monitoring-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add monitoring stack checks
	checks = append(checks, NewMonitoringStorageCheck())
	checks = append(checks, NewUserWorkloadMonitoringCheck())

	// Add logging checks
	checks = append(checks, NewLoggingInstallCheck())
	checks = append(checks, NewLoggingHealthCheck())
	checks = append(checks, NewLoggingStorageCheck())
	checks = append(checks, NewLoggingForwarderCheck())
	checks = append(checks, NewLoggingPlacementCheck())

	// Add service monitor check
	checks = append(checks, NewServiceMonitorCheck())

	return checks
}
