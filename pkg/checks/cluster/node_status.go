/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for node status. It:

- Verifies that all nodes in the cluster are in the Ready state
- Identifies nodes with issues requiring investigation
- Examines node conditions that could impact workload placement
- Provides recommendations for addressing node problems
- Helps ensure a healthy compute foundation for the cluster

This check is fundamental to overall cluster health, ensuring that the compute layer is functioning properly.
*/

package cluster

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// NodeStatusCheck checks if all nodes are ready
type NodeStatusCheck struct {
	healthcheck.BaseCheck
}

// NewNodeStatusCheck creates a new node status check
func NewNodeStatusCheck() *NodeStatusCheck {
	return &NodeStatusCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"node-status",
			"Node Status",
			"Checks if all nodes are ready",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *NodeStatusCheck) Run() (healthcheck.Result, error) {
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
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to retrieve nodes",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving nodes: %v", err)
	}

	// Check node status
	var notReadyNodes []string

	for _, node := range nodes.Items {
		nodeReady := false

		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				nodeReady = true
				break
			}
		}

		if !nodeReady {
			notReadyNodes = append(notReadyNodes, node.Name)
		}
	}

	// Get the output of 'oc get nodes' for detailed information
	detailedOut, err := utils.RunCommand("oc", "get", "nodes")

	// Format the detailed output with proper AsciiDoc formatting
	var formattedDetailedOut strings.Builder
	formattedDetailedOut.WriteString("=== Node Status Analysis ===\n\n")

	// Add node overview with proper formatting
	if err == nil && strings.TrimSpace(detailedOut) != "" {
		formattedDetailedOut.WriteString("Node Overview:\n[source, bash]\n----\n")
		formattedDetailedOut.WriteString(detailedOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Node Overview: No information available\n\n")
	}

	// Get extended node information including roles, versions, etc.
	extendedNodeInfo, _ := utils.RunCommand("oc", "get", "nodes", "-o", "wide")
	if strings.TrimSpace(extendedNodeInfo) != "" {
		formattedDetailedOut.WriteString("Detailed Node Information:\n[source, bash]\n----\n")
		formattedDetailedOut.WriteString(extendedNodeInfo)
		formattedDetailedOut.WriteString("\n----\n\n")
	}

	// Add node conditions information
	formattedDetailedOut.WriteString("=== Node Status Summary ===\n\n")
	formattedDetailedOut.WriteString(fmt.Sprintf("Total Nodes: %d\n", len(nodes.Items)))
	formattedDetailedOut.WriteString(fmt.Sprintf("Ready Nodes: %d\n", len(nodes.Items)-len(notReadyNodes)))
	formattedDetailedOut.WriteString(fmt.Sprintf("Not Ready Nodes: %d\n\n", len(notReadyNodes)))

	if len(notReadyNodes) > 0 {
		formattedDetailedOut.WriteString("Nodes Not Ready:\n")
		for _, nodeName := range notReadyNodes {
			formattedDetailedOut.WriteString(fmt.Sprintf("- %s\n", nodeName))
		}
		formattedDetailedOut.WriteString("\n")

		// Add troubleshooting guidance for not ready nodes
		formattedDetailedOut.WriteString("=== Troubleshooting Guidance ===\n\n")
		formattedDetailedOut.WriteString("For not ready nodes, check the following:\n\n")
		formattedDetailedOut.WriteString("1. Check node logs: `oc adm node-logs <node-name>`\n")
		formattedDetailedOut.WriteString("2. Check node diagnostics: `oc debug node/<node-name>`\n")
		formattedDetailedOut.WriteString("3. Check kubelet status on the node\n")
		formattedDetailedOut.WriteString("4. Verify network connectivity to the node\n")
		formattedDetailedOut.WriteString("5. Check for resource constraints (disk space, memory)\n\n")
	}

	if len(notReadyNodes) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			fmt.Sprintf("All %d nodes are ready", len(nodes.Items)),
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// Create result with not ready nodes information
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusCritical,
		fmt.Sprintf("%d nodes are not ready: %s", len(notReadyNodes), strings.Join(notReadyNodes, ", ")),
		types.ResultKeyRequired,
	)

	result.AddRecommendation("Investigate why the nodes are not ready")
	result.AddRecommendation("Check node logs using 'oc adm node-logs <node-name>'")
	result.AddRecommendation("Check node diagnostics using 'oc debug node/<node-name>'")

	result.Detail = formattedDetailedOut.String()
	return result, nil
}
