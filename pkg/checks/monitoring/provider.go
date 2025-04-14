package monitoring

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all monitoring-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Following the order in the PDF (Op-Ready category):

	// Logging forwarders for audit logs
	checks = append(checks, NewLoggingForwardersOpsCheck())

	// Logging forwarders for application logs
	checks = append(checks, NewLoggingForwarderCheck())

	// Logging installation and health
	checks = append(checks, NewLoggingInstallCheck())
	checks = append(checks, NewLoggingHealthCheck())

	// Logging placement
	checks = append(checks, NewLoggingPlacementCheck())

	// Logging storage
	checks = append(checks, NewLoggingStorageCheck())

	// Service monitors
	checks = append(checks, NewServiceMonitorCheck())

	// Alerts forwarding
	// checks = append(checks, NewAlertsForwardingCheck())
	// Mentioned in PDF but not implemented

	// Monitoring storage
	checks = append(checks, NewMonitoringStorageCheck())

	// User workload monitoring
	checks = append(checks, NewUserWorkloadMonitoringCheck())

	return checks
}
