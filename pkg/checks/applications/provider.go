package applications

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
)

// NewProbesCheck creates a new probes check
func NewProbesCheck() *ProbesCheck {
	return &ProbesCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"application-probes",
			"Application Probes",
			"Checks if applications have readiness and liveness probes configured",
			types.CategoryAppDev, // Changed from CategoryApplications
		),
	}
}

// NewResourceQuotasCheck creates a new resource quotas check
func NewResourceQuotasCheck() *ResourceQuotasCheck {
	return &ResourceQuotasCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"resource-quotas",
			"Resource Quotas",
			"Checks if resource quotas and limits are configured",
			types.CategoryAppDev, // Changed from CategoryApplications
		),
	}
}

// NewEmptyDirVolumeCheck creates a new empty directory volume check
func NewEmptyDirVolumeCheck() *EmptyDirVolumeCheck {
	return &EmptyDirVolumeCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"emptydir-volumes",
			"EmptyDir Volumes",
			"Checks for applications using emptyDir volumes, which are ephemeral and not recommended for persistent data",
			types.CategoryAppDev, // Changed from CategoryApplications
		),
	}
}

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
