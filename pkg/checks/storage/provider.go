package storage

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all storage-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add storage class check
	checks = append(checks, NewStorageClassCheck())

	// Add persistent volume check
	checks = append(checks, NewPersistentVolumeCheck())

	// Add storage performance check
	checks = append(checks, NewStoragePerformanceCheck())

	return checks
}
