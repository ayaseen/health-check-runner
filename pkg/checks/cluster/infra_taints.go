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
		result.Detail = fmt.Sprintf("Nodes without taints:\n%s\n\nTaint information:\n%s\n\nNodes:\n%s",
			strings.Join(nodesWithoutTaints, "\n"), taintInfo, detailedOut)
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
		result.Detail = fmt.Sprintf("Nodes with insufficient taints:\n%s\n\nTaint information:\n%s\n\nNodes:\n%s",
			strings.Join(nodesWithInsufficientTaints, "\n"), taintInfo, detailedOut)
		return result, nil
	}

	// All checks passed
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("All %d infrastructure nodes are properly tainted", len(nodes.Items)),
		types.ResultKeyNoChange,
	)
	result.Detail = fmt.Sprintf("Taint information:\n%s\n\nNodes:\n%s", taintInfo, detailedOut)
	return result, nil
}
