/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for ingress controller placement. It:

- Verifies if the ingress controller is placed on infrastructure nodes
- Checks for proper node selector configuration
- Ensures router pods are running on the designated nodes
- Provides recommendations for optimal ingress traffic handling
- Helps ensure proper separation of infrastructure components

This check helps maintain a properly architected environment where infrastructure components like ingress controllers are appropriately placed.
*/

package networking

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// IngressControllerPlacementCheck checks if the ingress controller is properly placed on infrastructure nodes
type IngressControllerPlacementCheck struct {
	healthcheck.BaseCheck
}

// NewIngressControllerPlacementCheck creates a new ingress controller placement check
func NewIngressControllerPlacementCheck() *IngressControllerPlacementCheck {
	return &IngressControllerPlacementCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"ingress-controller-placement",
			"Ingress Controller Placement",
			"Checks if the ingress controller is properly placed on infrastructure nodes",
			types.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *IngressControllerPlacementCheck) Run() (healthcheck.Result, error) {
	// Get the ingress controller configuration
	out, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator", "-o", "jsonpath={.spec.nodePlacement}")

	// Check if node placement is configured
	nodePlacementConfigured := err == nil && strings.TrimSpace(out) != "" && strings.TrimSpace(out) != "{}"

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator", "-o", "yaml")
	if err != nil {
		detailedOut = "Failed to get detailed ingress controller configuration"
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailedOut strings.Builder
	formattedDetailedOut.WriteString("=== Ingress Controller Placement Analysis ===\n\n")

	// Add main ingress controller configuration with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailedOut.WriteString("Ingress Controller Configuration:\n[source, yaml]\n----\n")
		formattedDetailedOut.WriteString(detailedOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Ingress Controller Configuration: No information available\n\n")
	}

	// Check if it's specifically placed on infra nodes
	infraPlacementConfig, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator", "-o", "jsonpath={.spec.nodePlacement.nodeSelector.matchLabels}")
	placedOnInfraNodes := err == nil && strings.Contains(infraPlacementConfig, "node-role.kubernetes.io/infra")

	// Add node selector information
	if strings.TrimSpace(infraPlacementConfig) != "" {
		formattedDetailedOut.WriteString("Node Selector Configuration:\n[source, text]\n----\n")
		formattedDetailedOut.WriteString(infraPlacementConfig)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Node Selector Configuration: No information available\n\n")
	}

	// Check if the router pods are actually running on infra nodes
	routerPodsOut, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-ingress", "-l", "ingresscontroller.operator.openshift.io/deployment-ingresscontroller=default", "-o", "wide")

	// Add router pods information
	if strings.TrimSpace(routerPodsOut) != "" {
		formattedDetailedOut.WriteString("Router Pods:\n[source, bash]\n----\n")
		formattedDetailedOut.WriteString(routerPodsOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Router Pods: No information available\n\n")
	}

	// Check if infra nodes exist in the cluster
	infraNodesOut, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=")

	// Add infrastructure nodes information
	if strings.TrimSpace(infraNodesOut) != "" && !strings.Contains(infraNodesOut, "No resources found") {
		formattedDetailedOut.WriteString("Infrastructure Nodes:\n[source, bash]\n----\n")
		formattedDetailedOut.WriteString(infraNodesOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Infrastructure Nodes: No nodes with 'node-role.kubernetes.io/infra=' label found\n\n")
	}

	infraNodesExist := err == nil && !strings.Contains(infraNodesOut, "No resources found")

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// If there are no infra nodes, recommend creating them
	if !infraNodesExist {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No infrastructure nodes found for ingress controller placement",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Create dedicated infrastructure nodes for router and other infrastructure components")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/machine_management/index#creating-infrastructure-machinesets", version))
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// If node placement isn't configured
	if !nodePlacementConfigured {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Ingress controller node placement is not configured",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure the ingress controller to be placed on infrastructure nodes")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/networking/index#nw-ingress-controller-configuration-parameters_configuring-ingress", version))
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// If not placed on infra nodes specifically
	if !placedOnInfraNodes {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Ingress controller is not configured to be placed on infrastructure nodes",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure the ingress controller to be placed on infrastructure nodes using the node-role.kubernetes.io/infra label")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/networking/index#nw-ingress-controller-configuration-parameters_configuring-ingress", version))
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// Check if the pods are actually running on infra nodes
	allPodsOnInfraNodes := true
	if strings.Contains(routerPodsOut, "NAME") {
		// Get list of infra node names
		infraNodeNames, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=", "-o", "jsonpath={.items[*].metadata.name}")
		if err == nil && infraNodeNames != "" {
			// Create a map for faster lookup
			infraNodes := make(map[string]bool)
			for _, nodeName := range strings.Split(infraNodeNames, " ") {
				if nodeName != "" {
					infraNodes[nodeName] = true
				}
			}

			// Check each router pod's node
			monPodLines := strings.Split(routerPodsOut, "\n")
			nonInfraNodePods := []string{}

			if len(monPodLines) > 1 { // Skip header line
				for _, line := range monPodLines[1:] {
					if line == "" {
						continue
					}
					fields := strings.Fields(line)
					if len(fields) >= 7 { // Check if the line has enough fields
						podName := fields[0]
						nodeName := fields[6]

						if !infraNodes[nodeName] {
							allPodsOnInfraNodes = false
							nonInfraNodePods = append(nonInfraNodePods, fmt.Sprintf("%s on %s", podName, nodeName))
						}
					}
				}
			}

			// Add information about pods not on infra nodes if any
			if len(nonInfraNodePods) > 0 {
				formattedDetailedOut.WriteString("Pods not on infrastructure nodes:\n")
				for _, pod := range nonInfraNodePods {
					formattedDetailedOut.WriteString("- " + pod + "\n")
				}
				formattedDetailedOut.WriteString("\n")
			}
		}
	}

	if !allPodsOnInfraNodes {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Some ingress controller pods are not running on infrastructure nodes",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Verify the node selector configuration and that all router pods are placed on infrastructure nodes")
		result.AddRecommendation("Check if the nodes have the correct labels and if there are any taints preventing scheduling")
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// All checks passed
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Ingress controller is properly placed on infrastructure nodes",
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailedOut.String()
	return result, nil
}
