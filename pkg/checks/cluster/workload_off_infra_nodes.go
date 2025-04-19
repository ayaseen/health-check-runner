/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for workload placement relative to infrastructure nodes. It:

- Identifies user workloads running on infrastructure nodes
- Verifies proper separation of applications from infrastructure components
- Examines pod scheduling across different node types
- Provides recommendations for proper workload isolation
- Helps maintain dedicated resources for infrastructure components

This check ensures that infrastructure nodes remain dedicated to their intended purpose without resource competition from application workloads.
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

// WorkloadOffInfraNodesCheck checks if workloads are scheduled on infrastructure nodes
type WorkloadOffInfraNodesCheck struct {
	healthcheck.BaseCheck
}

// NewWorkloadOffInfraNodesCheck creates a new workload off infrastructure nodes check
func NewWorkloadOffInfraNodesCheck() *WorkloadOffInfraNodesCheck {
	return &WorkloadOffInfraNodesCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"workload-off-infra-nodes",
			"Workloads on Infrastructure Nodes",
			"Checks if user workloads are scheduled on infrastructure nodes",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *WorkloadOffInfraNodesCheck) Run() (healthcheck.Result, error) {
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
	infraNodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
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

	// If no infrastructure nodes exist, this check is not applicable
	if len(infraNodes.Items) == 0 {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"No dedicated infrastructure nodes found in the cluster",
			types.ResultKeyNotApplicable,
		), nil
	}

	// Get all pods in user namespaces and check if they are scheduled on infrastructure nodes
	var podsOnInfraNodes []string
	var namespaces []string

	// Get all namespaces
	allNamespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to retrieve namespaces",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving namespaces: %v", err)
	}

	// Filter out system namespaces
	for _, ns := range allNamespaces.Items {
		if !strings.HasPrefix(ns.Name, "openshift-") &&
			ns.Name != "default" && ns.Name != "kube-system" &&
			ns.Name != "kube-public" && ns.Name != "kube-node-lease" {
			namespaces = append(namespaces, ns.Name)
		}
	}

	// Create a map of infrastructure node names for faster lookup
	infraNodeMap := make(map[string]bool)
	for _, node := range infraNodes.Items {
		infraNodeMap[node.Name] = true
	}

	// Check pods in user namespaces
	for _, ns := range namespaces {
		pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Non-critical error, we can continue with other namespaces
			continue
		}

		for _, pod := range pods.Items {
			// Skip pods that are being terminated
			if pod.DeletionTimestamp != nil {
				continue
			}

			// Check if the pod is running on an infrastructure node
			if infraNodeMap[pod.Spec.NodeName] {
				podsOnInfraNodes = append(podsOnInfraNodes,
					fmt.Sprintf("- Pod '%s' in namespace '%s' is running on infrastructure node '%s'",
						pod.Name, pod.Namespace, pod.Spec.NodeName))
			}
		}
	}

	// Get detailed node information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=", "-o", "wide")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed infrastructure node information"
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Workload Placement Analysis ===\n\n")

	// Add infrastructure nodes information with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut.WriteString("Infrastructure Nodes:\n[source, bash]\n----\n")
		formattedDetailOut.WriteString(detailedOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Infrastructure Nodes: No information available\n\n")
	}

	// Add node count information
	formattedDetailOut.WriteString(fmt.Sprintf("Total Infrastructure Nodes: %d\n", len(infraNodes.Items)))
	formattedDetailOut.WriteString(fmt.Sprintf("Total User Namespaces: %d\n\n", len(namespaces)))

	// Add user workloads section
	formattedDetailOut.WriteString("=== User Workloads on Infrastructure Nodes ===\n\n")
	if len(podsOnInfraNodes) > 0 {
		formattedDetailOut.WriteString(fmt.Sprintf("Found %d Application/User workloads running on infrastructure nodes:\n\n", len(podsOnInfraNodes)))
		for _, podInfo := range podsOnInfraNodes {
			formattedDetailOut.WriteString(podInfo + "\n")
		}
		formattedDetailOut.WriteString("\n")
	} else {
		formattedDetailOut.WriteString("No user workloads found running on infrastructure nodes (Good)\n\n")
	}

	// Add best practices section
	formattedDetailOut.WriteString("=== Best Practices ===\n\n")
	formattedDetailOut.WriteString("Infrastructure nodes should be dedicated to infrastructure components such as:\n")
	formattedDetailOut.WriteString("- Registry\n")
	formattedDetailOut.WriteString("- Router\n")
	formattedDetailOut.WriteString("- Monitoring\n")
	formattedDetailOut.WriteString("- Logging\n")
	formattedDetailOut.WriteString("- Metrics\n\n")
	formattedDetailOut.WriteString("User workloads should be scheduled on worker nodes to ensure proper resource allocation and prevent interference with critical infrastructure components.\n\n")

	// If no user workloads are running on infrastructure nodes, the check passes
	if len(podsOnInfraNodes) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"No user workloads are running on infrastructure nodes",
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Create result with workloads on infrastructure nodes information
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		fmt.Sprintf("%d user workloads are running on infrastructure nodes", len(podsOnInfraNodes)),
		types.ResultKeyRequired,
	)

	result.AddRecommendation("Infrastructure nodes should be dedicated to infrastructure components")
	result.AddRecommendation("Add taints to infrastructure nodes to prevent user workloads from running on them")
	result.AddRecommendation("Consider moving these workloads to worker nodes")

	result.Detail = formattedDetailOut.String()
	return result, nil
}
