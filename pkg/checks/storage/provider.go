/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file acts as a provider for storage-related health checks. It includes:

- A registry of all available storage health checks
- Functions to retrieve and initialize storage checks
- Organization of checks related to storage classes, persistent volumes, and storage performance

The provider ensures that all storage-related health checks are properly registered and available for execution by the main runner.
*/

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
