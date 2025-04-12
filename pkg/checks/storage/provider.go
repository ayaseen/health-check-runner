package storage

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all storage-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add storage-related checks here
	// In a real implementation, there would be actual storage checks
	// For example:
	// checks = append(checks, NewStorageClassCheck())
	// checks = append(checks, NewPersistentVolumeCheck())
	// checks = append(checks, NewStoragePerformanceCheck())

	return checks
}
