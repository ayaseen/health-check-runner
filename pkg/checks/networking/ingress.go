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

	// If there are no issues, return OK result
	if len(issues) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"Ingress controller is properly configured",
			types.ResultKeyNoChange,
		)
		result.Detail = detailedOut
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

	// Add the issues as details
	detailedResult := "Issues:\n" + strings.Join(issues, "\n") + "\n\n" + detailedOut

	// Add the recommendations
	for _, rec := range recommendedActions {
		result.AddRecommendation(rec)
	}

	result.Detail = detailedResult

	return result, nil
}
