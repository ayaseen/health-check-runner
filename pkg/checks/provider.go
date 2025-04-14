package checks

import (
	"github.com/ayaseen/health-check-runner/pkg/checks/applications"
	"github.com/ayaseen/health-check-runner/pkg/checks/cluster"
	"github.com/ayaseen/health-check-runner/pkg/checks/monitoring"
	"github.com/ayaseen/health-check-runner/pkg/checks/networking"
	"github.com/ayaseen/health-check-runner/pkg/checks/security"
	"github.com/ayaseen/health-check-runner/pkg/checks/storage"
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetOpenShiftChecks returns all OpenShift-specific health checks
func GetOpenShiftChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add cluster checks first (includes Infra checks)
	checks = append(checks, cluster.GetChecks()...)

	// Add networking checks
	checks = append(checks, networking.GetChecks()...)

	// Add storage checks
	checks = append(checks, storage.GetChecks()...)

	// Add security checks
	checks = append(checks, security.GetChecks()...)

	// Add monitoring checks (Op-Ready in the PDF)
	checks = append(checks, monitoring.GetChecks()...)

	return checks
}

// GetApplicationChecks returns all application-specific health checks
func GetApplicationChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add application checks
	checks = append(checks, applications.GetChecks()...)

	return checks
}

// GetStorageChecks returns all storage-related health checks
func GetStorageChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add storage checks
	checks = append(checks, storage.GetChecks()...)

	return checks
}

// GetAllChecks returns all available health checks
func GetAllChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add cluster checks first (includes Infra checks)
	checks = append(checks, cluster.GetChecks()...)

	// Add networking checks
	checks = append(checks, networking.GetChecks()...)

	// Add storage checks
	checks = append(checks, storage.GetChecks()...)

	// Add application checks (App Dev in the PDF)
	checks = append(checks, applications.GetChecks()...)

	// Add security checks
	checks = append(checks, security.GetChecks()...)

	// Add monitoring checks (Op-Ready in the PDF)
	checks = append(checks, monitoring.GetChecks()...)

	return checks
}
