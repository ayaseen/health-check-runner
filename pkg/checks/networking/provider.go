package networking

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all networking-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add CNI network plugin check
	checks = append(checks, NewCNINetworkPluginCheck())

	// Add network policy check
	checks = append(checks, NewNetworkPolicyCheck())

	// Add ingress controller check
	checks = append(checks, NewIngressControllerCheck())

	// Add new ingress controller placement check
	checks = append(checks, NewIngressControllerPlacementCheck())

	// Add new ingress controller replica check
	checks = append(checks, NewIngressControllerReplicaCheck())

	//// Add new ingress controller type check
	//checks = append(checks, NewIngressControllerTypeCheck())
	//
	//// Add new default ingress certificate check
	//checks = append(checks, NewDefaultIngressCertificateCheck())

	return checks
}
