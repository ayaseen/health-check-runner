/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for control plane node scheduling. It:

- Verifies if control plane nodes are properly protected from regular workloads
- Checks for appropriate taints on master/control nodes
- Examines node scheduling configuration for control plane protection
- Provides recommendations for proper control plane isolation
- Helps maintain control plane stability by preventing resource contention

This check helps ensure that critical control plane components have dedicated resources without competition from application workloads.
*/

package cluster

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// ControlNodeSchedulableCheck checks if control plane nodes are marked as unschedulable for workloads
type ControlNodeSchedulableCheck struct {
	healthcheck.BaseCheck
}

// NewControlNodeSchedulableCheck creates a new control node schedulable check
func NewControlNodeSchedulableCheck() *ControlNodeSchedulableCheck {
	return &ControlNodeSchedulableCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"control-node-schedulable",
			"Control Plane Node Schedulability",
			"Checks if control plane nodes are marked as unschedulable for regular workloads",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *ControlNodeSchedulableCheck) Run() (healthcheck.Result, error) {
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

	// Get the list of nodes
	ctx := context.Background()
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/master=",
	})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to retrieve control plane nodes",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving control plane nodes: %v", err)
	}

	// Check if any control plane nodes are schedulable
	var schedulableControlNodes []string

	for _, node := range nodes.Items {
		if !node.Spec.Unschedulable {
			// Check if there's a taint that prevents regular workloads
			hasTaint := false
			for _, taint := range node.Spec.Taints {
				if taint.Key == "node-role.kubernetes.io/master" &&
					(taint.Effect == "NoSchedule" || taint.Effect == "NoExecute") {
					hasTaint = true
					break
				}
			}

			if !hasTaint {
				schedulableControlNodes = append(schedulableControlNodes, node.Name)
			}
		}
	}

	// Get detailed node information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/master=", "-o", "wide")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed control plane node information"
	}

	// Get taint information for all control plane nodes
	taintInfo, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/master=", "-o", "jsonpath={range .items[*]}{.metadata.name}{\": \"}{.spec.taints}{\"\\n\"}{end}")
	if err != nil {
		taintInfo = "Failed to get taint information"
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Control Plane Node Schedulability Analysis ===\n\n")

	// Add control plane node information with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut.WriteString("Control Plane Nodes:\n[source, bash]\n----\n")
		formattedDetailOut.WriteString(detailedOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Control Plane Nodes: No information available\n\n")
	}

	// Add taint information with proper formatting
	if strings.TrimSpace(taintInfo) != "" {
		formattedDetailOut.WriteString("Taint Information:\n[source, yaml]\n----\n")
		formattedDetailOut.WriteString(taintInfo)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Taint Information: No information available\n\n")
	}

	// Add schedulability summary
	formattedDetailOut.WriteString("=== Schedulability Summary ===\n\n")
	formattedDetailOut.WriteString(fmt.Sprintf("Total Control Plane Nodes: %d\n", len(nodes.Items)))
	formattedDetailOut.WriteString(fmt.Sprintf("Schedulable Control Plane Nodes: %d\n\n", len(schedulableControlNodes)))

	if len(schedulableControlNodes) > 0 {
		formattedDetailOut.WriteString("Control plane nodes allowing regular workloads:\n")
		for _, nodeName := range schedulableControlNodes {
			formattedDetailOut.WriteString(fmt.Sprintf("- %s\n", nodeName))
		}
		formattedDetailOut.WriteString("\n")
	}

	// Add best practices section
	formattedDetailOut.WriteString("=== Best Practices ===\n\n")
	formattedDetailOut.WriteString("Control plane nodes should be dedicated to control plane components to ensure stability and performance of the Kubernetes control plane.\n\n")
	formattedDetailOut.WriteString("To prevent scheduling of regular workloads on control plane nodes, either:\n")
	formattedDetailOut.WriteString("1. Add the NoSchedule taint: 'node-role.kubernetes.io/master=:NoSchedule'\n")
	formattedDetailOut.WriteString("2. Mark the node as unschedulable using 'oc adm cordon <node-name>'\n\n")
	formattedDetailOut.WriteString("This ensures that only pods with matching tolerations (typically control plane components) will be scheduled on these nodes.\n\n")

	if len(schedulableControlNodes) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"All control plane nodes are properly configured to prevent regular workloads",
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Create result with schedulable control node information
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		fmt.Sprintf("%d control plane nodes allow scheduling of regular workloads: %s",
			len(schedulableControlNodes), strings.Join(schedulableControlNodes, ", ")),
		types.ResultKeyRecommended,
	)

	result.AddRecommendation("Control plane nodes should be dedicated to control plane components for better reliability")
	result.AddRecommendation("Add NoSchedule taints to control plane nodes using 'oc adm taint nodes <node-name> node-role.kubernetes.io/master=:NoSchedule'")
	result.AddRecommendation("Alternatively, mark control plane nodes as unschedulable using 'oc adm cordon <node-name>'")

	result.Detail = formattedDetailOut.String()
	return result, nil
}
