package security

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all security-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Following the order in the PDF:

	// Default Security Context Constraint (SCC)
	checks = append(checks, NewClusterDefaultSCCCheck())

	// Default Project Template
	checks = append(checks, NewDefaultProjectTemplateCheck())

	// Self-Provisioner role
	checks = append(checks, NewSelfProvisionerCheck())

	// Kubeadmin user
	checks = append(checks, NewKubeadminUserCheck())

	// Identity Provider checks
	checks = append(checks, NewIdentityProviderCheck())

	// ETCD backup and encryption checks
	checks = append(checks, NewEtcdBackupCheck())
	checks = append(checks, NewEtcdEncryptionCheck())

	// Elevated privileges - in App Dev category in PDF but implementation is in security
	checks = append(checks, NewElevatedPrivilegesCheck())

	return checks
}
