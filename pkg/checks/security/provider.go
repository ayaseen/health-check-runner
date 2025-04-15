/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file acts as a provider for security-related health checks. It includes:

- A registry of all available security health checks
- Functions to retrieve and initialize security checks
- Organization of checks related to SCCs, authentication, encryption, and privileges
- Registration of checks for security best practices

The provider ensures that all security-related health checks are properly registered and available for execution by the main runner.
*/

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
