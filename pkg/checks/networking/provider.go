package networking

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
)

// NewCNINetworkPluginCheck creates a new CNI network plugin check
func NewCNINetworkPluginCheck() *CNINetworkPluginCheck {
	return &CNINetworkPluginCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"cni-network-plugin",
			"CNI Network Plugin",
			"Checks if the cluster is using the recommended CNI network plugin",
			types.CategoryNetwork, // Changed from CategoryNetworking
		),
	}
}

// NewNetworkPolicyCheck creates a new network policy check
func NewNetworkPolicyCheck() *NetworkPolicyCheck {
	return &NetworkPolicyCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"network-policy",
			"Network Policy",
			"Checks if network policies are configured for traffic control",
			types.CategoryNetwork, // Changed from CategoryNetworking
		),
	}
}

// IngressControllerCheck uses new category
func NewIngressControllerCheck() *IngressControllerCheck {
	return &IngressControllerCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"ingress-controller",
			"Ingress Controller",
			"Checks if the ingress controller is properly configured",
			types.CategoryNetwork, // Changed from CategoryNetworking
		),
	}
}

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
