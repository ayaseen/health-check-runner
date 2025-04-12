package networking

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
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
			healthcheck.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *IngressControllerCheck) Run() (healthcheck.Result, error) {
	// This is a comprehensive check that evaluates multiple aspects of the ingress controller

	// Check if the ingress controller is the default type
	typeResult, err := c.checkControllerType()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check ingress controller type",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking ingress controller type: %v", err)
	}

	// Check ingress controller placement
	placementResult, err := c.checkControllerPlacement()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check ingress controller placement",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking ingress controller placement: %v", err)
	}

	// Check ingress controller replica count
	replicaResult, err := c.checkControllerReplicas()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check ingress controller replicas",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking ingress controller replicas: %v", err)
	}

	// Check if the ingress controller certificate is properly configured
	certResult, err := c.checkControllerCertificate()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check ingress controller certificate",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking ingress controller certificate: %v", err)
	}

	// Combine the results of each sub-check
	issues := []string{}
	recommendedActions := []string{}

	// Add type check result
	if typeResult.Status != healthcheck.StatusOK {
		issues = append(issues, typeResult.Message)
		recommendedActions = append(recommendedActions, typeResult.Recommendations...)
	}

	// Add placement check result
	if placementResult.Status != healthcheck.StatusOK {
		issues = append(issues, placementResult.Message)
		recommendedActions = append(recommendedActions, placementResult.Recommendations...)
	}

	// Add replica check result
	if replicaResult.Status != healthcheck.StatusOK {
		issues = append(issues, replicaResult.Message)
		recommendedActions = append(recommendedActions, replicaResult.Recommendations...)
	}

	// Add certificate check result
	if certResult.Status != healthcheck.StatusOK {
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
			healthcheck.StatusOK,
			"Ingress controller is properly configured",
			healthcheck.ResultKeyNoChange,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// Create a result with issues and recommendations
	resultStatus := healthcheck.StatusWarning
	resultKey := healthcheck.ResultKeyRecommended

	// If there are critical issues, set status to Critical
	for _, subResult := range []healthcheck.Result{typeResult, placementResult, replicaResult, certResult} {
		if subResult.Status == healthcheck.StatusCritical {
			resultStatus = healthcheck.StatusCritical
			resultKey = healthcheck.ResultKeyRequired
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

// checkControllerType checks if the ingress controller is the default type
func (c *IngressControllerCheck) checkControllerType() (healthcheck.Result, error) {
	out, err := utils.RunCommand("oc", "get", "deployment/router-default", "-n", "openshift-ingress",
		"-o", `jsonpath={.metadata.labels.ingresscontroller\.operator\.openshift\.io\/owning-ingresscontroller}`)
	if err != nil {
		return healthcheck.NewResult(
			"ingress-controller-type",
			healthcheck.StatusCritical,
			"Failed to get ingress controller type",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting ingress controller type: %v", err)
	}

	ingressType := strings.TrimSpace(out)

	if ingressType == "default" {
		return healthcheck.NewResult(
			"ingress-controller-type",
			healthcheck.StatusOK,
			"Default OpenShift ingress controller is in use",
			healthcheck.ResultKeyNoChange,
		), nil
	}

	result := healthcheck.NewResult(
		"ingress-controller-type",
		healthcheck.StatusWarning,
		fmt.Sprintf("Non-default ingress controller type is in use: %s", ingressType),
		healthcheck.ResultKeyAdvisory,
	)

	result.AddRecommendation("Ensure the non-default ingress controller meets your requirements")

	return result, nil
}

// checkControllerPlacement checks if the ingress controller is placed on infrastructure nodes
func (c *IngressControllerCheck) checkControllerPlacement() (healthcheck.Result, error) {
	out, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator",
		"-o", `jsonpath={.spec.nodePlacement.nodeSelector.matchLabels}`)
	if err != nil {
		return healthcheck.NewResult(
			"ingress-controller-placement",
			healthcheck.StatusCritical,
			"Failed to get ingress controller placement",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting ingress controller placement: %v", err)
	}

	placement := strings.TrimSpace(out)

	if strings.Contains(placement, "node-role.kubernetes.io/infra") {
		return healthcheck.NewResult(
			"ingress-controller-placement",
			healthcheck.StatusOK,
			"Ingress controller is placed on infrastructure nodes",
			healthcheck.ResultKeyNoChange,
		), nil
	}

	result := healthcheck.NewResult(
		"ingress-controller-placement",
		healthcheck.StatusWarning,
		"Ingress controller is not placed on dedicated infrastructure nodes",
		healthcheck.ResultKeyRecommended,
	)

	result.AddRecommendation("Configure the ingress controller to run on dedicated infrastructure nodes")
	result.AddRecommendation("Follow the documentation at https://docs.openshift.com/container-platform/latest/networking/ingress-operator.html#nw-ingress-controller-configuration-parameters_configuring-ingress")

	return result, nil
}

// checkControllerReplicas checks if the ingress controller has the recommended number of replicas
func (c *IngressControllerCheck) checkControllerReplicas() (healthcheck.Result, error) {
	out, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator",
		"-o", `jsonpath='{.spec.replicas}'`)
	if err != nil {
		return healthcheck.NewResult(
			"ingress-controller-replicas",
			healthcheck.StatusCritical,
			"Failed to get ingress controller replicas",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting ingress controller replicas: %v", err)
	}

	// Remove the surrounding quotes from the output
	replicaStr := strings.Trim(strings.TrimSpace(out), "'")

	if replicaStr == "" {
		// No replica count specified, likely using default (auto-scaling)
		return healthcheck.NewResult(
			"ingress-controller-replicas",
			healthcheck.StatusOK,
			"Ingress controller is using default replica configuration",
			healthcheck.ResultKeyNoChange,
		), nil
	}

	replicas, err := strconv.Atoi(replicaStr)
	if err != nil {
		return healthcheck.NewResult(
			"ingress-controller-replicas",
			healthcheck.StatusCritical,
			"Failed to parse ingress controller replica count",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error parsing ingress controller replicas: %v", err)
	}

	// Recommended minimum is 3 for high availability
	if replicas >= 3 {
		return healthcheck.NewResult(
			"ingress-controller-replicas",
			healthcheck.StatusOK,
			fmt.Sprintf("Ingress controller has sufficient replicas: %d", replicas),
			healthcheck.ResultKeyNoChange,
		), nil
	}

	result := healthcheck.NewResult(
		"ingress-controller-replicas",
		healthcheck.StatusWarning,
		fmt.Sprintf("Ingress controller has insufficient replicas: %d (recommended: >= 3)", replicas),
		healthcheck.ResultKeyRecommended,
	)

	result.AddRecommendation("Increase the number of ingress controller replicas to at least 3 for high availability")

	return result, nil
}

// checkControllerCertificate checks if the ingress controller certificate is properly configured
func (c *IngressControllerCheck) checkControllerCertificate() (healthcheck.Result, error) {
	// Check for custom domain certificate configuration
	out, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator",
		"-o", `jsonpath={.spec.defaultCertificate}`)
	if err != nil {
		return healthcheck.NewResult(
			"ingress-controller-certificate",
			healthcheck.StatusCritical,
			"Failed to get ingress controller certificate configuration",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting ingress controller certificate: %v", err)
	}

	certificate := strings.TrimSpace(out)

	// If a custom certificate is configured, the check passes
	if certificate != "" && certificate != "{}" {
		return healthcheck.NewResult(
			"ingress-controller-certificate",
			healthcheck.StatusOK,
			"Custom certificate is configured for the ingress controller",
			healthcheck.ResultKeyNoChange,
		), nil
	}

	// If no custom certificate is configured, recommend configuring one
	result := healthcheck.NewResult(
		"ingress-controller-certificate",
		healthcheck.StatusWarning,
		"Default (self-signed) certificate is used for the ingress controller",
		healthcheck.ResultKeyRecommended,
	)

	result.AddRecommendation("Configure a custom certificate for the ingress controller to avoid browser warnings")
	result.AddRecommendation("Follow the documentation at https://docs.openshift.com/container-platform/latest/security/certificates/replacing-default-ingress-certificate.html")

	return result, nil
}
