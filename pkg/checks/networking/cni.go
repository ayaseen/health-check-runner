/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for networking configurations. It includes:

- Checks for the recommended CNI network plugin (OVNKubernetes)
- Verification of network policy configuration
- Examination of network isolation between namespaces
- Recommendations for network security best practices
- Provider functions for registering networking checks

These checks help ensure optimal network performance and proper network security configurations in OpenShift clusters.
*/

package networking

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
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
			types.CategoryNetworking,
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
			types.StatusCritical,
			"Failed to get CNI network plugin type",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting CNI network plugin type: %v", err)
	}

	cniType := strings.TrimSpace(out)

	// Get the full network config for detailed information
	detailedOut, err := utils.RunCommand("oc", "get", "network.config", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed network configuration"
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailedOut string
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailedOut = fmt.Sprintf("Network Configuration:\n[source, yaml]\n----\n%s\n----\n\n", detailedOut)
	} else {
		formattedDetailedOut = "Network Configuration: No information available\n\n"
	}

	// Check if the CNI type is OVNKubernetes, which is the recommended type
	if cniType == "OVNKubernetes" {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			fmt.Sprintf("Cluster is using the recommended CNI network plugin: %s", cniType),
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailedOut
		return result, nil
	}

	// Create result with recommendation to use OVNKubernetes
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		fmt.Sprintf("Cluster is using CNI network plugin: %s (recommended: OVNKubernetes)", cniType),
		types.ResultKeyRequired,
	)

	// Add recommendations based on the current CNI type
	if cniType == "OpenShiftSDN" {
		result.AddRecommendation("Consider migrating to OVNKubernetes for better features and performance")
		result.AddRecommendation("Follow the migration guide at https://docs.openshift.com/container-platform/latest/networking/ovn_kubernetes_network_provider/migrate-from-openshift-sdn.html")
	} else {
		result.AddRecommendation("Consider using OVNKubernetes as the CNI network plugin")
	}

	result.Detail = formattedDetailedOut

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
			types.CategoryNetworking,
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
			result := healthcheck.NewResult(
				c.ID(),
				types.StatusWarning,
				"No network policies found in the cluster",
				types.ResultKeyRecommended,
			)
			result.Detail = "No network policies configured"
			return result, nil
		}

		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get network policies",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting network policies: %v", err)
	}

	// Format the detailed output with proper spacing
	var formattedDetailedOut string
	if strings.TrimSpace(out) != "" {
		formattedDetailedOut = fmt.Sprintf("Network Policies:\n[source, bash]\n----\n%s\n----\n\n", out)
	} else {
		formattedDetailedOut = "Network Policies: No information available\n\n"
	}

	// Check if any network policies exist
	lines := strings.Split(out, "\n")
	if len(lines) <= 1 { // Only header line or empty
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No network policies found in the cluster",
			types.ResultKeyRecommended,
		)
		result.Detail = formattedDetailedOut
		return result, nil
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
		types.StatusOK,
		fmt.Sprintf("Found %d network policies in the cluster", policyCount),
		types.ResultKeyNoChange,
	)

	result.Detail = formattedDetailedOut

	return result, nil
}

// GetNetworkingChecks returns networking-related health checks - renamed to avoid conflict
func GetNetworkingChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add CNI network plugin check
	checks = append(checks, NewCNINetworkPluginCheck())

	// Add network policy check
	checks = append(checks, NewNetworkPolicyCheck())

	return checks
}
