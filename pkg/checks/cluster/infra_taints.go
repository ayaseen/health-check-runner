/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for infrastructure node taints. It:

- Verifies if infrastructure nodes have appropriate taints
- Checks for missing or insufficient taints
- Examines taint effect types (NoSchedule, NoExecute, PreferNoSchedule)
- Provides recommendations for proper node isolation
- Helps ensure infrastructure nodes are dedicated to infrastructure workloads

This check complements the infrastructure node checks to ensure proper workload isolation and resource dedication.
*/

package cluster

import (
	"context"
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InfraTaintsCheck checks if infrastructure nodes are properly tainted
type InfraTaintsCheck struct {
	healthcheck.BaseCheck
}

// NewInfraTaintsCheck creates a new infrastructure taints check
func NewInfraTaintsCheck() *InfraTaintsCheck {
	return &InfraTaintsCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"infra-taints",
			"Infrastructure Taints",
			"Checks if infrastructure nodes are properly tainted to prevent regular workloads",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *InfraTaintsCheck) Run() (healthcheck.Result, error) {
	// Get Kubernetes clientset
	clientset, err := utils.GetClientSet()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get Kubernetes client",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting Kubernetes client: %v", err)
	}

	// Get the list of infrastructure nodes
	ctx := context.Background()
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/infra=",
	})

	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to retrieve infrastructure nodes",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving infrastructure nodes: %v", err)
	}

	// If no infrastructure nodes found, return NotApplicable
	if len(nodes.Items) == 0 {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"No infrastructure nodes found in the cluster",
			types.ResultKeyNotApplicable,
		), nil
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=", "-o", "wide")
	if err != nil {
		detailedOut = "Failed to get detailed infrastructure node information"
	}

	// Get taint information for all infra nodes
	taintInfo, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=", "-o", "jsonpath={range .items[*]}{.metadata.name}{\": \"}{.spec.taints}{\"\\n\"}{end}")
	if err != nil {
		taintInfo = "Failed to get taint information"
	}

	// Check if any infrastructure nodes are not tainted
	nodesWithoutTaints := []string{}
	nodesWithInsufficientTaints := []string{}

	for _, node := range nodes.Items {
		hasTaint := false
		hasStrongTaint := false

		for _, taint := range node.Spec.Taints {
			if taint.Key == "node-role.kubernetes.io/infra" {
				hasTaint = true
				if taint.Effect == "NoSchedule" || taint.Effect == "NoExecute" {
					hasStrongTaint = true
				}
			}
		}

		if !hasTaint {
			nodesWithoutTaints = append(nodesWithoutTaints, node.Name)
		} else if !hasStrongTaint {
			nodesWithInsufficientTaints = append(nodesWithInsufficientTaints, node.Name)
		}
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Infrastructure Node Taints Analysis ===\n\n")

	// Add infrastructure node information with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut.WriteString("Infrastructure Nodes:\n[source, bash]\n----\n")
		formattedDetailOut.WriteString(detailedOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Infrastructure Nodes: No information available\n\n")
	}

	// Add taint information with proper formatting
	if strings.TrimSpace(taintInfo) != "" {
		formattedDetailOut.WriteString("Taint Information:\n[source, yaml]\n----\n")
		formattedDetailOut.WriteString(taintInfo)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Taint Information: No information available\n\n")
	}

	// Add taint summary section
	formattedDetailOut.WriteString("=== Taint Status Summary ===\n\n")
	formattedDetailOut.WriteString(fmt.Sprintf("Total Infrastructure Nodes: %d\n", len(nodes.Items)))
	formattedDetailOut.WriteString(fmt.Sprintf("Nodes Without Taints: %d\n", len(nodesWithoutTaints)))
	formattedDetailOut.WriteString(fmt.Sprintf("Nodes With Insufficient Taints: %d\n\n", len(nodesWithInsufficientTaints)))

	if len(nodesWithoutTaints) > 0 {
		formattedDetailOut.WriteString("Nodes missing taints:\n")
		for _, nodeName := range nodesWithoutTaints {
			formattedDetailOut.WriteString(fmt.Sprintf("- %s\n", nodeName))
		}
		formattedDetailOut.WriteString("\n")
	}

	if len(nodesWithInsufficientTaints) > 0 {
		formattedDetailOut.WriteString("Nodes with insufficient taints (need NoSchedule or NoExecute):\n")
		for _, nodeName := range nodesWithInsufficientTaints {
			formattedDetailOut.WriteString(fmt.Sprintf("- %s\n", nodeName))
		}
		formattedDetailOut.WriteString("\n")
	}

	// Add best practices section
	formattedDetailOut.WriteString("=== Taint Best Practices ===\n\n")
	formattedDetailOut.WriteString("Infrastructure nodes should have appropriate taints to ensure only infrastructure workloads are scheduled on them.\n\n")
	formattedDetailOut.WriteString("Recommended taint for infrastructure nodes:\n")
	formattedDetailOut.WriteString("  node-role.kubernetes.io/infra:NoSchedule\n\n")
	formattedDetailOut.WriteString("This ensures regular application workloads will not be scheduled on infrastructure nodes,\n")
	formattedDetailOut.WriteString("while infrastructure components with matching tolerations can still be placed there.\n\n")

	// Evaluate taint configuration
	if len(nodesWithoutTaints) > 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("%d infrastructure nodes are not tainted", len(nodesWithoutTaints)),
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Add taints to infrastructure nodes to prevent regular workloads from being scheduled on them")
		result.AddRecommendation(fmt.Sprintf("Use 'oc adm taint nodes <node-name> node-role.kubernetes.io/infra=:NoSchedule'"))
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/nodes/scheduling-pods-to-specific-nodes", version))

		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	if len(nodesWithInsufficientTaints) > 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("%d infrastructure nodes have weak taints (PreferNoSchedule instead of NoSchedule)", len(nodesWithInsufficientTaints)),
			types.ResultKeyAdvisory,
		)
		result.AddRecommendation("Consider using stronger taint effects (NoSchedule or NoExecute) to ensure regular workloads aren't scheduled on infrastructure nodes")
		result.AddRecommendation(fmt.Sprintf("Use 'oc adm taint nodes <node-name> node-role.kubernetes.io/infra=:NoSchedule'"))

		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// All checks passed
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("All %d infrastructure nodes are properly tainted", len(nodes.Items)),
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailOut.String()
	return result, nil
}
