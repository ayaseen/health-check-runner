/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file acts as a provider for networking-related health checks. It includes:

- A registry of all available networking health checks
- Functions to retrieve and initialize networking checks
- Organization of checks related to CNI plugins, network policies, and ingress controllers
- Registration of checks for network connectivity and configuration

The provider ensures that all networking-related health checks are properly registered and available for execution by the main runner.
*/

package networking

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all networking-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Following the order in the PDF:

	// Ingress Controller Type
	checks = append(checks, NewIngressControllerTypeCheck())

	// Ingress Controller Placement
	checks = append(checks, NewIngressControllerPlacementCheck())

	// Ingress Controller Replica Count
	checks = append(checks, NewIngressControllerReplicaCheck())

	// Ingress Controller Certificate
	checks = append(checks, NewDefaultIngressCertificateCheck())

	// CNI Network Plugin
	checks = append(checks, NewCNINetworkPluginCheck())

	// Network Policy
	checks = append(checks, NewNetworkPolicyCheck())

	// The comprehensive ingress controller check - this might include all the above specific checks
	checks = append(checks, NewIngressControllerCheck())

	return checks
}
