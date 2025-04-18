/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for ingress controller replicas. It:

- Verifies if the ingress controller has sufficient replicas for high availability
- Checks for the recommended minimum of three replicas
- Examines the ingress controller deployment configuration
- Provides recommendations for proper ingress controller scaling
- Helps ensure resilient ingress traffic handling

This check helps maintain high availability for application routing in OpenShift clusters.
*/

package networking

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// IngressControllerReplicaCheck checks if the ingress controller has sufficient replicas
type IngressControllerReplicaCheck struct {
	healthcheck.BaseCheck
}

// NewIngressControllerReplicaCheck creates a new ingress controller replica check
func NewIngressControllerReplicaCheck() *IngressControllerReplicaCheck {
	return &IngressControllerReplicaCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"ingress-controller-replica",
			"Ingress Controller Replicas",
			"Checks if the ingress controller has sufficient replicas for high availability",
			types.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *IngressControllerReplicaCheck) Run() (healthcheck.Result, error) {
	// Get the replica count without quotes in the jsonpath expression
	out, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator",
		"-o", "jsonpath={.spec.replicas}")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get ingress controller replicas",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting ingress controller replicas: %v", err)
	}

	// Trim any whitespace from the output
	replicaStr := strings.TrimSpace(out)

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed ingress controller configuration"
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailedOut strings.Builder
	formattedDetailedOut.WriteString("=== Ingress Controller Replica Analysis ===\n\n")

	// Add main ingress controller configuration with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailedOut.WriteString("Ingress Controller Configuration:\n[source, yaml]\n----\n")
		formattedDetailedOut.WriteString(detailedOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Ingress Controller Configuration: No information available\n\n")
	}

	// Get router deployment information
	routerDeploymentOut, _ := utils.RunCommand("oc", "get", "deployment", "-n", "openshift-ingress", "-l", "ingresscontroller.operator.openshift.io/deployment-ingresscontroller=default")
	if strings.TrimSpace(routerDeploymentOut) != "" {
		formattedDetailedOut.WriteString("Router Deployment:\n[source, bash]\n----\n")
		formattedDetailedOut.WriteString(routerDeploymentOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	}

	// Get router pods information
	routerPodsOut, _ := utils.RunCommand("oc", "get", "pods", "-n", "openshift-ingress", "-l", "ingresscontroller.operator.openshift.io/deployment-ingresscontroller=default")
	if strings.TrimSpace(routerPodsOut) != "" {
		formattedDetailedOut.WriteString("Router Pods:\n[source, bash]\n----\n")
		formattedDetailedOut.WriteString(routerPodsOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	}

	if replicaStr == "" {
		// No replica count specified, likely using default (auto-scaling)
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Ingress controller is using default replica configuration, which may not be optimal",
			types.ResultKeyAdvisory,
		)
		result.AddRecommendation("Configure a specific replica count for better control over the ingress controller scaling")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/networking/index#configuring-ingress", version))

		formattedDetailedOut.WriteString("No explicit replica count found in the ingress controller configuration.\n")
		formattedDetailedOut.WriteString("The ingress controller is likely using the default auto-scaling configuration.\n\n")

		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	replicas, err := strconv.Atoi(replicaStr)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to parse ingress controller replica count",
			types.ResultKeyRequired,
		), fmt.Errorf("error parsing ingress controller replicas: %v", err)
	}

	// Add replica count information to the detail output
	formattedDetailedOut.WriteString(fmt.Sprintf("Configured Replica Count: %d\n\n", replicas))
	formattedDetailedOut.WriteString("Recommendation: Production environments should have at least 3 replicas for high availability.\n\n")

	// Recommended minimum is 3 for high availability
	if replicas >= 3 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			fmt.Sprintf("Ingress controller has sufficient replicas: %d", replicas),
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		fmt.Sprintf("Ingress controller has insufficient replicas: %d (recommended: >= 3)", replicas),
		types.ResultKeyRequired,
	)

	result.AddRecommendation("Increase the number of ingress controller replicas to at least 3 for high availability")
	result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/networking/index#configuring-ingress", version))
	result.Detail = formattedDetailedOut.String()

	return result, nil
}
