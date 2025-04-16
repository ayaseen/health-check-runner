/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for ingress controller type. It:

- Verifies if the ingress controller is of the recommended type
- Examines the OpenShift router deployment to determine the controller type
- Checks if the default ingress controller is being used
- Provides recommendations if non-default controller types are detected
- Helps ensure proper application routing configuration

This check helps maintain standard and supported ingress controller configurations in OpenShift clusters, ensuring reliable application access.
*/

package networking

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// IngressControllerTypeCheck checks if the ingress controller is the default type
type IngressControllerTypeCheck struct {
	healthcheck.BaseCheck
}

// NewIngressControllerTypeCheck creates a new ingress controller type check
func NewIngressControllerTypeCheck() *IngressControllerTypeCheck {
	return &IngressControllerTypeCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"ingress-controller-type",
			"Ingress Controller Type",
			"Checks if the ingress controller is of the recommended type",
			types.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *IngressControllerTypeCheck) Run() (healthcheck.Result, error) {
	out, err := utils.RunCommand("oc", "get", "deployment/router-default", "-n", "openshift-ingress",
		"-o", `jsonpath={.metadata.labels.ingresscontroller\.operator\.openshift\.io/owning-ingresscontroller}`)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get ingress controller type",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting ingress controller type: %v", err)
	}

	ingressType := strings.TrimSpace(out)

	// Get detailed information about the ingress controller deployment
	deploymentOut, err := utils.RunCommand("oc", "get", "deployment/router-default", "-n", "openshift-ingress", "-o", "yaml")

	// Create the exact format for the detail output with proper spacing
	var formattedDetailedOut strings.Builder
	formattedDetailedOut.WriteString("=== Ingress Controller Type Analysis ===\n\n")

	// Add ingress controller type information
	formattedDetailedOut.WriteString(fmt.Sprintf("Ingress Controller Type: %s\n\n", ingressType))

	// Add ingress controller deployment information
	if strings.TrimSpace(deploymentOut) != "" {
		formattedDetailedOut.WriteString("Ingress Controller Deployment:\n[source, yaml]\n----\n")
		formattedDetailedOut.WriteString(deploymentOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Ingress Controller Deployment: No information available\n\n")
	}

	// Get router pods information
	routerPodsOut, _ := utils.RunCommand("oc", "get", "pods", "-n", "openshift-ingress", "-l", "ingresscontroller.operator.openshift.io/deployment-ingresscontroller=default")
	if strings.TrimSpace(routerPodsOut) != "" {
		formattedDetailedOut.WriteString("Router Pods:\n[source, bash]\n----\n")
		formattedDetailedOut.WriteString(routerPodsOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	}

	// Get available ingress controllers
	ingressControllersOut, _ := utils.RunCommand("oc", "get", "ingresscontroller", "-n", "openshift-ingress-operator")
	if strings.TrimSpace(ingressControllersOut) != "" {
		formattedDetailedOut.WriteString("Available Ingress Controllers:\n[source, bash]\n----\n")
		formattedDetailedOut.WriteString(ingressControllersOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	}

	if ingressType == "default" {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"Default OpenShift ingress controller is in use",
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		fmt.Sprintf("Non-default ingress controller type is in use: %s", ingressType),
		types.ResultKeyAdvisory,
	)

	result.AddRecommendation("Ensure the non-default ingress controller meets your requirements")
	result.Detail = formattedDetailedOut.String()

	return result, nil
}
