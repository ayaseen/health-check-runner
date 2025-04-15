/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file acts as a provider for monitoring-related health checks. It includes:

- A registry of all available monitoring and logging health checks
- Functions to retrieve and initialize monitoring checks
- Organization of checks related to alerts, logs, and metrics
- Registration of checks for monitoring storage and configuration

The provider ensures that all monitoring-related health checks are properly registered and available for execution by the main runner.
*/

package monitoring

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all monitoring-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Following the order in the PDF (Op-Ready category):

	// Enhanced monitoring stack configuration check
	checks = append(checks, NewMonitoringStackConfigCheck())

	// Logging forwarders for operations (infrastructure and audit) logs
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
	checks = append(checks, NewAlertsForwardingCheck())

	// Monitoring storage
	checks = append(checks, NewMonitoringStorageCheck())

	// User workload monitoring
	checks = append(checks, NewUserWorkloadMonitoringCheck())

	return checks
}
