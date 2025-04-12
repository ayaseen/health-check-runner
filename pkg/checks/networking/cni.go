package networking

import (
	"fmt"
	"strings"

	"gitlab.consulting.redhat.com/meta/health-check-runner/pkg/healthcheck"
	"gitlab.consulting.redhat.com/meta/health-check-runner/pkg/utils"
)

// CNINetworkPluginCheck checks the CNI plugin configuration
type CNINetworkPluginCheck struct {
	healthcheck.BaseCheck
}

// NewCNINetworkPluginCheck creates a new CNI network plugin check
func NewCNINetworkPluginCheck() *CNINetworkPluginCheck {
	return &CNINetworkPluginCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"cni-network-plugin",
			"CNI Network Plugin",
			"Checks if the cluster is using the recommended CNI network plugin",
			healthcheck.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *CNINetworkPluginCheck) Run() (healthcheck.Result, error) {
	// Get the CNI network plugin type
	out, err := utils.RunCommand("oc", "get", "network.config", "-o", "jsonpath={.items[*].spec.networkType}")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get CNI network plugin type",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting CNI network plugin type: %v", err)
	}

	cniType := strings.TrimSpace(out)

	// Get the full network config for detailed information
	detailedOut, err := utils.RunCommand("oc", "get", "network.config", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed network configuration"
	}

	// Check if the CNI type is OVNKubernetes, which is the recommended type
	if cniType == "OVNKubernetes" {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			fmt.Sprintf("Cluster is using the recommended CNI network plugin: %s", cniType),
			healthcheck.ResultKeyNoChange,
		).WithDetail(detailedOut), nil
	}

	// Create result with recommendation to use OVNKubernetes
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusWarning,
		fmt.Sprintf("Cluster is using CNI network plugin: %s (recommended: OVNKubernetes)", cniType),
		healthcheck.ResultKeyRecommended,
	)

	// Add recommendations based on the current CNI type
	if cniType == "OpenShiftSDN" {
		result.AddRecommendation("Consider migrating to OVNKubernetes for better features and performance")
		result.AddRecommendation("Follow the migration guide at https://docs.openshift.com/container-platform/latest/networking/ovn_kubernetes_network_provider/migrate-from-openshift-sdn.html")
	} else {
		result.AddRecommendation("Consider using OVNKubernetes as the CNI network plugin")
	}

	result.WithDetail(detailedOut)

	return result, nil
}

// NetworkPolicyCheck checks if network policies are configured
type NetworkPolicyCheck struct {
	healthcheck.BaseCheck
}

// NewNetworkPolicyCheck creates a new network policy check
func NewNetworkPolicyCheck() *NetworkPolicyCheck {
	return &NetworkPolicyCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"network-policy",
			"Network Policy",
			"Checks if network policies are configured for traffic control",
			healthcheck.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *NetworkPolicyCheck) Run() (healthcheck.Result, error) {
	// Get network policies across all namespaces
	out, err := utils.RunCommand("oc", "get", "netpol", "--all-namespaces")
	if err != nil {
		// This might not be a critical error, as it could just mean no network policies exist
		if strings.Contains(err.Error(), "No resources found") {
			// No network policies found
			return healthcheck.NewResult(
				c.ID(),
				healthcheck.StatusWarning,
				"No network policies found in the cluster",
				healthcheck.ResultKeyRecommended,
			).WithDetail("No network policies configured"), nil
		}

		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get network policies",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting network policies: %v", err)
	}

	// Check if any network policies exist
	lines := strings.Split(out, "\n")
	if len(lines) <= 1 { // Only header line or empty
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"No network policies found in the cluster",
			healthcheck.ResultKeyRecommended,
		).WithDetail("No network policies configured"), nil
	}

	// Count the number of network policies (excluding header line)
	policyCount := len(lines) - 1
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			policyCount--
		}
	}

	// Create result based on the number of policies found
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		fmt.Sprintf("Found %d network policies in the cluster", policyCount),
		healthcheck.ResultKeyNoChange,
	)

	result.WithDetail(out)

	return result, nil
}

// GetChecks returns all networking-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add CNI network plugin check
	checks = append(checks, NewCNINetworkPluginCheck())

	// Add network policy check
	checks = append(checks, NewNetworkPolicyCheck())

	// Add other networking checks here
	checks = append(checks, NewIngressControllerCheck())

	return checks
}
