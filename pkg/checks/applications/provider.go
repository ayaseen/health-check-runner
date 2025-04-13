package applications

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
)

// GetChecks returns all application-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add probes check with updated category
	checks = append(checks, &ProbesCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"application-probes",
			"Application Probes",
			"Checks if applications have readiness and liveness probes configured",
			types.CategoryAppDev, // Changed from CategoryApplications
		),
	})

	// Add resource quotas check with updated category
	checks = append(checks, &ResourceQuotasCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"resource-quotas",
			"Resource Quotas",
			"Checks if resource quotas and limits are configured",
			types.CategoryAppDev, // Changed from CategoryApplications
		),
	})

	// Add empty dir check with updated category
	checks = append(checks, &EmptyDirVolumeCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"emptydir-volumes",
			"EmptyDir Volumes",
			"Checks for applications using emptyDir volumes, which are ephemeral and not recommended for persistent data",
			types.CategoryAppDev, // Changed from CategoryApplications
		),
	})

	return checks
}
