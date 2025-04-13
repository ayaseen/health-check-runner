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

	// Add ingress controller checks
	checks = append(checks, NewIngressControllerCheck())

	return checks
}
