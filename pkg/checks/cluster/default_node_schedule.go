/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for default node scheduling configuration. It:

- Examines node role labeling and tainting
- Verifies proper node selector configuration for namespaces
- Checks for custom scheduler configuration
- Ensures nodes have appropriate roles assigned
- Provides recommendations for workload placement control

This check helps maintain a well-organized cluster with proper workload placement rules across different node types.
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

// DefaultNodeScheduleCheck checks if the default scheduling configuration for nodes is appropriate
type DefaultNodeScheduleCheck struct {
	healthcheck.BaseCheck
}

// NewDefaultNodeScheduleCheck creates a new default node schedule check
func NewDefaultNodeScheduleCheck() *DefaultNodeScheduleCheck {
	return &DefaultNodeScheduleCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"default-node-schedule",
			"Default Node Schedule",
			"Checks if the default scheduling configuration for nodes is appropriate",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *DefaultNodeScheduleCheck) Run() (healthcheck.Result, error) {
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

	// Check if the scheduler configuration
	schedulerConfig, err := utils.RunCommand("oc", "get", "configmap", "scheduler-config", "-n", "openshift-kube-scheduler", "-o", "yaml")

	customSchedulerConfig := err == nil && strings.Contains(schedulerConfig, "policy:")

	// Get detailed information about nodes for the report
	detailedOut, err := utils.RunCommand("oc", "get", "nodes", "-o", "wide")
	if err != nil {
		detailedOut = "Failed to get detailed node information"
	}

	// Check if all nodes are properly labeled
	workloadNodes := 0
	infraNodes := 0
	controlNodes := 0
	nodesWithoutRole := []string{}

	for _, node := range nodes.Items {
		// Check for role labels
		_, isControl := node.Labels["node-role.kubernetes.io/master"]
		_, isInfra := node.Labels["node-role.kubernetes.io/infra"]
		_, isWorker := node.Labels["node-role.kubernetes.io/worker"]

		if isControl {
			controlNodes++
		}
		if isInfra {
			infraNodes++
		}
		if isWorker {
			workloadNodes++
		}

		// Check if node has no role
		if !isControl && !isInfra && !isWorker {
			nodesWithoutRole = append(nodesWithoutRole, node.Name)
		}
	}

	// Check for node selectors in namespaces
	namespaceNodeSelectors, err := utils.RunCommand("oc", "get", "namespaces", "-o", "jsonpath={range .items[*]}{.metadata.name}{\": \"}{.metadata.annotations.openshift\\.io/node-selector}{\"\\n\"}{end}")

	customNamespaceNodeSelectorConfigured := err == nil && !strings.Contains(namespaceNodeSelectors, ": \n")

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Node Scheduling Configuration Analysis ===\n\n")

	// Add node list with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut.WriteString("Cluster Nodes:\n[source, bash]\n----\n")
		formattedDetailOut.WriteString(detailedOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Cluster Nodes: No information available\n\n")
	}

	// Add node role summary
	formattedDetailOut.WriteString("=== Node Role Distribution ===\n\n")
	formattedDetailOut.WriteString(fmt.Sprintf("Total Nodes: %d\n", len(nodes.Items)))
	formattedDetailOut.WriteString(fmt.Sprintf("Control Plane Nodes: %d\n", controlNodes))
	formattedDetailOut.WriteString(fmt.Sprintf("Infrastructure Nodes: %d\n", infraNodes))
	formattedDetailOut.WriteString(fmt.Sprintf("Worker Nodes: %d\n", workloadNodes))
	formattedDetailOut.WriteString(fmt.Sprintf("Nodes Without Role: %d\n\n", len(nodesWithoutRole)))

	if len(nodesWithoutRole) > 0 {
		formattedDetailOut.WriteString("Nodes without any role label:\n")
		for _, nodeName := range nodesWithoutRole {
			formattedDetailOut.WriteString(fmt.Sprintf("- %s\n", nodeName))
		}
		formattedDetailOut.WriteString("\n")
	}

	// Add namespace node selector information with proper formatting
	if strings.TrimSpace(namespaceNodeSelectors) != "" {
		formattedDetailOut.WriteString("Namespace Node Selectors:\n[source, yaml]\n----\n")
		formattedDetailOut.WriteString(namespaceNodeSelectors)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Namespace Node Selectors: No information available\n\n")
	}

	// Add scheduler configuration information with proper formatting
	if strings.TrimSpace(schedulerConfig) != "" {
		formattedDetailOut.WriteString("Scheduler Configuration:\n[source, yaml]\n----\n")
		formattedDetailOut.WriteString(schedulerConfig)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Scheduler Configuration: Using default configuration\n\n")
	}

	// Add best practices section
	formattedDetailOut.WriteString("=== Best Practices ===\n\n")
	formattedDetailOut.WriteString("1. All nodes should have appropriate role labels\n")
	formattedDetailOut.WriteString("2. Configure namespace node selectors to control workload placement\n")
	formattedDetailOut.WriteString("3. Use custom scheduler configuration for fine-grained control\n")
	formattedDetailOut.WriteString("4. Separate workloads, infrastructure, and control plane functions\n\n")

	// Evaluate node scheduling configuration
	if len(nodesWithoutRole) > 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("%d nodes do not have any role label", len(nodesWithoutRole)),
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Label all nodes with appropriate role labels (worker, infra, or master)")
		result.AddRecommendation(fmt.Sprintf("Use 'oc label node <node-name> node-role.kubernetes.io/worker='"))
		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Check if we have a good distribution of node roles
	if workloadNodes == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No nodes are labeled as worker nodes",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Label appropriate nodes with the worker role")
		result.AddRecommendation(fmt.Sprintf("Use 'oc label node <node-name> node-role.kubernetes.io/worker='"))
		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Check if custom namespace node selectors are configured
	if !customNamespaceNodeSelectorConfigured {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No custom namespace node selectors are configured",
			types.ResultKeyAdvisory,
		)
		result.AddRecommendation("Consider configuring namespace node selectors to control workload placement")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/nodes/index#nodes-scheduler-node-selectors", version))
		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Check if custom scheduler configuration is in place
	if !customSchedulerConfig {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"Default node scheduling configuration is acceptable",
			types.ResultKeyNoChange,
		)
		result.AddRecommendation("For more advanced scheduling control, consider configuring custom scheduler policies")
		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// All checks passed with optimal configuration
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Node scheduling is well configured with custom policies",
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailOut.String()
	return result, nil
}
