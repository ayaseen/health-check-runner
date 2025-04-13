package networking

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
)

// GetChecks returns all networking-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add CNI network plugin check with updated category
	checks = append(checks, &CNINetworkPluginCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"cni-network-plugin",
			"CNI Network Plugin",
			"Checks if the cluster is using the recommended CNI network plugin",
			types.CategoryNetwork, // Changed from CategoryNetworking
		),
	})

	// Add network policy check with updated category
	checks = append(checks, &NetworkPolicyCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"network-policy",
			"Network Policy",
			"Checks if network policies are configured for traffic control",
			types.CategoryNetwork, // Changed from CategoryNetworking
		),
	})

	// Add ingress controller check with updated category
	checks = append(checks, &IngressControllerCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"ingress-controller",
			"Ingress Controller",
			"Checks if the ingress controller is properly configured",
			types.CategoryNetwork, // Changed from CategoryNetworking
		),
	})

	return checks
}
