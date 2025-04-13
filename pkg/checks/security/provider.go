package security

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all security-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add SCC check
	checks = append(checks, NewClusterDefaultSCCCheck())

	// Add elevated privileges check (consolidated version)
	checks = append(checks, NewElevatedPrivilegesCheck())

	// Add ETCD security checks
	checks = append(checks, NewEtcdEncryptionCheck())
	checks = append(checks, NewEtcdBackupCheck())
	checks = append(checks, NewEtcdHealthCheck())

	// Add default project template check
	checks = append(checks, NewDefaultProjectTemplateCheck())

	// Add kubeadmin user check
	checks = append(checks, NewKubeadminUserCheck())

	// Add identity provider check
	checks = append(checks, NewIdentityProviderCheck())

	// Add self-provisioner check
	checks = append(checks, NewSelfProvisionerCheck())

	return checks
}
