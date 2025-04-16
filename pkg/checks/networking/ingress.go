/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements comprehensive health checks for ingress controller configuration. It:

- Coordinates multiple ingress-related checks (type, placement, replica, certificate)
- Aggregates findings from individual checks into a comprehensive assessment
- Provides consolidated recommendations for ingress configuration
- Identifies critical ingress issues that could impact application accessibility
- Helps ensure reliable and secure application routing

This check provides a holistic view of ingress controller health and configuration in OpenShift clusters.
*/

package networking

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// IngressControllerCheck checks the ingress controller configuration
type IngressControllerCheck struct {
	healthcheck.BaseCheck
}

// NewIngressControllerCheck creates a new ingress controller check
func NewIngressControllerCheck() *IngressControllerCheck {
	return &IngressControllerCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"ingress-controller",
			"Ingress Controller",
			"Checks if the ingress controller is properly configured",
			types.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *IngressControllerCheck) Run() (healthcheck.Result, error) {
	// Use individual checks instead of internal methods
	typeCheck := NewIngressControllerTypeCheck()
	placementCheck := NewIngressControllerPlacementCheck()
	replicaCheck := NewIngressControllerReplicaCheck()
	certCheck := NewDefaultIngressCertificateCheck()

	// Run each check
	typeResult, typeErr := typeCheck.Run()
	placementResult, placementErr := placementCheck.Run()
	replicaResult, replicaErr := replicaCheck.Run()
	certResult, certErr := certCheck.Run()

	// Check for critical errors in any check
	if typeErr != nil || placementErr != nil || replicaErr != nil || certErr != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to check ingress controller configuration",
			types.ResultKeyRequired,
		), fmt.Errorf("error checking ingress controller configuration")
	}

	// Combine the results of each sub-check
	issues := []string{}
	recommendedActions := []string{}

	// Add type check result
	if typeResult.Status != types.StatusOK {
		issues = append(issues, typeResult.Message)
		recommendedActions = append(recommendedActions, typeResult.Recommendations...)
	}

	// Add placement check result
	if placementResult.Status != types.StatusOK {
		issues = append(issues, placementResult.Message)
		recommendedActions = append(recommendedActions, placementResult.Recommendations...)
	}

	// Add replica check result
	if replicaResult.Status != types.StatusOK {
		issues = append(issues, replicaResult.Message)
		recommendedActions = append(recommendedActions, replicaResult.Recommendations...)
	}

	// Add certificate check result
	if certResult.Status != types.StatusOK {
		issues = append(issues, certResult.Message)
		recommendedActions = append(recommendedActions, certResult.Recommendations...)
	}

	// Get detailed information about the ingress controller
	detailedOut, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed ingress controller configuration"
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailedOut strings.Builder
	formattedDetailedOut.WriteString("=== Comprehensive Ingress Controller Analysis ===\n\n")

	// Add main ingress controller configuration with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailedOut.WriteString("Ingress Controller Configuration:\n[source, yaml]\n----\n")
		formattedDetailedOut.WriteString(detailedOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Ingress Controller Configuration: No information available\n\n")
	}

	// Add individual check results
	formattedDetailedOut.WriteString("=== Individual Check Results ===\n\n")
	formattedDetailedOut.WriteString("Type Check: " + typeResult.Message + "\n\n")
	formattedDetailedOut.WriteString("Placement Check: " + placementResult.Message + "\n\n")
	formattedDetailedOut.WriteString("Replica Check: " + replicaResult.Message + "\n\n")
	formattedDetailedOut.WriteString("Certificate Check: " + certResult.Message + "\n\n")

	// Add router pods information
	routerPodsOut, _ := utils.RunCommand("oc", "get", "pods", "-n", "openshift-ingress", "-l", "ingresscontroller.operator.openshift.io/deployment-ingresscontroller=default", "-o", "wide")
	if strings.TrimSpace(routerPodsOut) != "" {
		formattedDetailedOut.WriteString("Router Pods:\n[source, bash]\n----\n")
		formattedDetailedOut.WriteString(routerPodsOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	}

	// If there are no issues, return OK result
	if len(issues) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"Ingress controller is properly configured",
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// Create a result with issues and recommendations
	resultStatus := types.StatusWarning
	resultKey := types.ResultKeyRecommended

	// If there are critical issues, set status to Critical
	for _, subResult := range []healthcheck.Result{typeResult, placementResult, replicaResult, certResult} {
		if subResult.Status == types.StatusCritical {
			resultStatus = types.StatusCritical
			resultKey = types.ResultKeyRequired
			break
		}
	}

	result := healthcheck.NewResult(
		c.ID(),
		resultStatus,
		fmt.Sprintf("Ingress controller has %d configuration issues", len(issues)),
		resultKey,
	)

	// Add issues section
	formattedDetailedOut.WriteString("=== Configuration Issues ===\n\n")
	for _, issue := range issues {
		formattedDetailedOut.WriteString("- " + issue + "\n")
	}
	formattedDetailedOut.WriteString("\n")

	// Add the recommendations
	for _, rec := range recommendedActions {
		result.AddRecommendation(rec)
	}

	result.Detail = formattedDetailedOut.String()
	return result, nil
}
