package performance

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all performance-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add all performance checks here
	checks = append(checks, NewClusterPerformanceCheck())

	// Add user workload performance check
	checks = append(checks, NewUserWorkloadPerformanceCheck())

	// Add additional performance checks as they are developed
	// checks = append(checks, NewOtherPerformanceCheck())

	return checks
}
