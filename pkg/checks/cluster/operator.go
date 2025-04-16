/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for cluster operators. It:

- Verifies if all cluster operators are available and not degraded
- Identifies operators with issues that need attention
- Provides detailed operator status information
- Recommends troubleshooting steps for problematic operators
- Helps ensure the core cluster functionality is working correctly

This check is critical for overall cluster health, as operators manage key components of the OpenShift platform.
*/

package cluster

import (
	"context"
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// ClusterOperatorsCheck checks if all cluster operators are available
type ClusterOperatorsCheck struct {
	healthcheck.BaseCheck
}

// NewClusterOperatorsCheck creates a new cluster operators check
func NewClusterOperatorsCheck() *ClusterOperatorsCheck {
	return &ClusterOperatorsCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"cluster-operators",
			"Cluster Operators",
			"Checks if all cluster operators are available",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *ClusterOperatorsCheck) Run() (healthcheck.Result, error) {
	// Get Kubernetes config
	config, err := utils.GetClusterConfig()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get cluster config",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting cluster config: %v", err)
	}

	// Create OpenShift client
	client, err := versioned.NewForConfig(config)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to create OpenShift client",
			types.ResultKeyRequired,
		), fmt.Errorf("error creating OpenShift client: %v", err)
	}

	// Get the list of cluster operators
	ctx := context.Background()
	cos, err := client.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to retrieve cluster operators",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving cluster operators: %v", err)
	}

	// Check if all cluster operators are available
	allAvailable := true
	var unavailableOps []string

	for _, co := range cos.Items {
		available := false

		for _, condition := range co.Status.Conditions {
			if condition.Type == configv1.OperatorAvailable && condition.Status == configv1.ConditionTrue {
				available = true
				break
			}
		}

		if !available {
			allAvailable = false
			unavailableOps = append(unavailableOps, co.Name)
		}
	}

	// Get the output of 'oc get co' for detailed information
	detailedOut, err := utils.RunCommand("oc", "get", "co")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed operator status"
	}

	// Format the detailed output with proper AsciiDoc formatting
	var formattedDetailedOut strings.Builder
	formattedDetailedOut.WriteString("=== Cluster Operators Status ===\n\n")

	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailedOut.WriteString("Cluster Operators Overview:\n[source, bash]\n----\n")
		formattedDetailedOut.WriteString(detailedOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Cluster Operators Overview: No information available\n\n")
	}

	// Add operator analysis section
	formattedDetailedOut.WriteString("=== Operator Analysis ===\n\n")
	formattedDetailedOut.WriteString(fmt.Sprintf("Total Cluster Operators: %d\n", len(cos.Items)))

	if len(unavailableOps) > 0 {
		formattedDetailedOut.WriteString("\nUnavailable Operators:\n")
		for _, op := range unavailableOps {
			formattedDetailedOut.WriteString(fmt.Sprintf("- %s\n", op))

			// Try to get more details for unavailable operators
			opDetails, _ := utils.RunCommand("oc", "describe", "co", op)
			if strings.TrimSpace(opDetails) != "" {
				formattedDetailedOut.WriteString(fmt.Sprintf("\nDetails for operator %s:\n[source, yaml]\n----\n%s\n----\n\n", op, opDetails))
			}
		}
	} else {
		formattedDetailedOut.WriteString("\nAll operators are available.\n")
	}
	formattedDetailedOut.WriteString("\n")

	if allAvailable {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"All cluster operators are available",
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// Create result with unavailable operators information
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusCritical,
		fmt.Sprintf("Some cluster operators are not available: %s", strings.Join(unavailableOps, ", ")),
		types.ResultKeyRequired,
	)

	result.AddRecommendation("Investigate why the operators are not available")
	result.AddRecommendation("Check operator logs using 'oc logs deployment/<operator-name> -n <operator-namespace>'")
	result.AddRecommendation("Consult the OpenShift documentation or Red Hat support")

	result.Detail = formattedDetailedOut.String()
	return result, nil
}
