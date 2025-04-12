package applications

import (
	"gitlab.consulting.redhat.com/meta/health-check-runner/pkg/healthcheck"
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

	// Add additional application checks here

	return checks
}
