/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file acts as a provider for application-related health checks. It includes:

- A registry of all available application health checks
- Functions to retrieve and initialize application checks
- Organization of checks related to application configurations and best practices
- Registration of checks for probes, resource quotas, and volume usage

The provider ensures that all application-related health checks are properly registered and available for execution by the main runner.
*/

package applications

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all application-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add probes check
	checks = append(checks, NewProbesCheck())

	// Add resource quotas check
	checks = append(checks, NewResourceQuotasCheck())

	// Add empty dir check
	checks = append(checks, NewEmptyDirVolumeCheck())

	// Add new limit range check
	checks = append(checks, NewLimitRangeCheck())

	return checks
}
