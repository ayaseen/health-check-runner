package security

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all security-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Get base security checks from the renamed function in scc.go
	checks = append(checks, GetSecurityChecks()...)

	// Add privileged containers check - our new implementation with different name
	checks = append(checks, NewPrivilegedContainersCheck())

	// Add default project template check - moved from cluster package
	checks = append(checks, NewDefaultProjectTemplateCheck())

	// Add kubeadmin user check - moved from cluster package
	checks = append(checks, NewKubeadminUserCheck())

	// Add identity provider check
	checks = append(checks, NewIdentityProviderCheck())

	// Add self-provisioner check
	checks = append(checks, NewSelfProvisionerCheck())

	return checks
}
