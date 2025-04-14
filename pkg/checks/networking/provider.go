package networking

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all networking-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Following the order in the PDF:

	// Ingress Controller Type
	// checks = append(checks, NewIngressControllerTypeCheck())
	// Not implemented yet, but mentioned in the PDF

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
