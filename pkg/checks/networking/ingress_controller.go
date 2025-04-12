package networking

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// IngressControllerCheck is a comprehensive check for the ingress controller
type IngressControllerCheck struct {
	healthcheck.BaseCheck
}

// NewIngressControllerCheck creates a new ingress controller check
func NewIngressControllerCheck() *IngressControllerCheck {
	return &IngressControllerCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"ingress-controller",
			"Ingress Controller",
			"Checks the ingress controller configuration",
			healthcheck.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *IngressControllerCheck) Run() (healthcheck.Result, error) {
	// This check will evaluate multiple aspects of the ingress controller
	// 1. Type (default or not)
	// 2. Placement (infrastructure nodes)
	// 3. Replica count (3 is recommended)
	// 4. Certificate (default or custom)

	// Get necessary information
	ingressType, typeErr := checkIngressControllerType()
	ingressPlacement, placementErr := checkIngressControllerPlacement()
	ingressReplicaCount, replicaErr := checkIngressControllerReplicas()

	// Get detailed info for pods
	ingressPodDetails, _ := utils.RunCommand("oc", "get", "pods", "-n", "openshift-ingress", "-o", "wide")

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version
	}

	// Create a slice to collect issues
	var issues []string
	var recommendations []string

	// Check type
	if typeErr == nil && ingressType != "default" {
		issues = append(issues, fmt.Sprintf("Non-default ingress controller type: %s", ingressType))
	}

	// Check placement
	hasInfraNodePlacement := false
	if placementErr == nil {
		hasInfraNodePlacement = strings.Contains(ingressPlacement, "node-role.kubernetes.io/infra")
	}

	if !hasInfraNodePlacement {
		issues = append(issues, "Ingress controller is not configured to run on infrastructure nodes")
		recommendations = append(recommendations, "Configure the ingress controller to run on infrastructure nodes")
		recommendations = append(recommendations, fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/networking/index#configuring-ingress", version))
	}

	// Check replica count
	hasThreeReplicas := false
	if replicaErr == nil {
		replicaStr := strings.Trim(ingressReplicaCount, "'")
		if replicaStr != "" {
			replicas, err := strconv.Atoi(replicaStr)
			if err == nil && replicas >= 3 {
				hasThreeReplicas = true
			}
		}
	}

	if !hasThreeReplicas {
		issues = append(issues, "Ingress controller has fewer than 3 replicas")
		recommendations = append(recommendations, "Configure at least 3 replicas for the ingress controller for high availability")
		recommendations = append(recommendations, fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/networking/index#configuring-ingress", version))
	}

	// Combine all information for detailed output
	var detailedOutput strings.Builder
	detailedOutput.WriteString("Ingress Controller Configuration:\n\n")
	detailedOutput.WriteString(fmt.Sprintf("Type: %s\n", ingressType))
	detailedOutput.WriteString(fmt.Sprintf("Node Placement: %s\n", ingressPlacement))
	detailedOutput.WriteString(fmt.Sprintf("Replica Count: %s\n\n", ingressReplicaCount))
	detailedOutput.WriteString("Ingress Pods:\n")
	detailedOutput.WriteString(ingressPodDetails)

	// Create result based on findings
	if len(issues) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			"Ingress controller is properly configured",
			healthcheck.ResultKeyNoChange,
		)
		result.Detail = detailedOutput.String()
		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusWarning,
		fmt.Sprintf("Ingress controller configuration has %d issues", len(issues)),
		healthcheck.ResultKeyRecommended,
	)

	for _, issue := range issues {
		result.Detail += issue + "\n"
	}

	for _, rec := range recommendations {
		result.AddRecommendation(rec)
	}

	result.Detail = detailedOutput.String()
	return result, nil
}

// Helper function to check the ingress controller type
func checkIngressControllerType() (string, error) {
	out, err := utils.RunCommand("oc", "get", "deployment/router-default", "-n", "openshift-ingress",
		"-o", "jsonpath={.metadata.labels.ingresscontroller\\.operator\\.openshift\\.io/owning-ingresscontroller}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Helper function to check the ingress controller placement
func checkIngressControllerPlacement() (string, error) {
	out, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator",
		"-o", "jsonpath={.spec.nodePlacement.nodeSelector.matchLabels}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Helper function to check the ingress controller replicas
func checkIngressControllerReplicas() (string, error) {
	out, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator",
		"-o", "jsonpath='{.spec.replicas}'")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// IngressControllerTypeCheck checks the type of ingress controller being used
type IngressControllerTypeCheck struct {
	healthcheck.BaseCheck
}

// NewIngressControllerTypeCheck creates a new ingress controller type check
func NewIngressControllerTypeCheck() *IngressControllerTypeCheck {
	return &IngressControllerTypeCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"ingress-controller-type",
			"Ingress Controller Type",
			"Checks if the default OpenShift ingress controller is in use",
			healthcheck.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *IngressControllerTypeCheck) Run() (healthcheck.Result, error) {
	// Get the ingress controller type
	ingressType, err := checkIngressControllerType()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check ingress controller type",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking ingress controller type: %v", err)
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version
	}

	// Check if it's the default type
	if ingressType != "default" {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			fmt.Sprintf("Non-default ingress controller type is in use: %s", ingressType),
			healthcheck.ResultKeyAdvisory,
		)
		result.AddRecommendation("Ensure the non-default ingress controller meets your requirements")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/networking/index#configuring-ingress", version))
		return result, nil
	}

	// Default type is in use
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		"Default OpenShift ingress controller is in use",
		healthcheck.ResultKeyNoChange,
	)
	return result, nil
}

// IngressControllerPlacementCheck checks if the ingress controller is placed on infrastructure nodes
type IngressControllerPlacementCheck struct {
	healthcheck.BaseCheck
}

// NewIngressControllerPlacementCheck creates a new ingress controller placement check
func NewIngressControllerPlacementCheck() *IngressControllerPlacementCheck {
	return &IngressControllerPlacementCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"ingress-controller-placement",
			"Ingress Controller Placement",
			"Checks if the ingress controller is placed on infrastructure nodes",
			healthcheck.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *IngressControllerPlacementCheck) Run() (healthcheck.Result, error) {
	// Get the ingress controller placement
	ingressPlacement, err := checkIngressControllerPlacement()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check ingress controller placement",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking ingress controller placement: %v", err)
	}

	// Get the actual placement of ingress pods
	ingressNodePlacement, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-ingress", "-o", "wide")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		ingressNodePlacement = "Failed to get ingress pod placement"
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version
	}

	// Check if it's placed on infrastructure nodes
	if !strings.Contains(ingressPlacement, "node-role.kubernetes.io/infra") {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"Ingress controller is not placed on infrastructure nodes",
			healthcheck.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure the ingress controller to run on infrastructure nodes")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/networking/index#configuring-ingress", version))
		result.Detail = ingressNodePlacement
		return result, nil
	}

	// Properly placed on infrastructure nodes
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		"Ingress controller is placed on infrastructure nodes",
		healthcheck.ResultKeyNoChange,
	)
	result.Detail = ingressNodePlacement
	return result, nil
}

// IngressControllerReplicaCheck checks if the ingress controller has the recommended number of replicas
type IngressControllerReplicaCheck struct {
	healthcheck.BaseCheck
}

// NewIngressControllerReplicaCheck creates a new ingress controller replica check
func NewIngressControllerReplicaCheck() *IngressControllerReplicaCheck {
	return &IngressControllerReplicaCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"ingress-controller-replica",
			"Ingress Controller Replica Count",
			"Checks if the ingress controller has the recommended number of replicas",
			healthcheck.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *IngressControllerReplicaCheck) Run() (healthcheck.Result, error) {
	// Get the ingress controller replicas
	ingressReplicaCount, err := checkIngressControllerReplicas()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check ingress controller replicas",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking ingress controller replicas: %v", err)
	}

	// Get the actual pods
	ingressPods, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-ingress")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		ingressPods = "Failed to get ingress pods"
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version
	}

	// Parse the replica count
	replicaStr := strings.Trim(ingressReplicaCount, "'")

	// If no replica count is configured, it's using defaults
	if replicaStr == "" {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"Ingress controller is using default replica configuration, which may not be optimal",
			healthcheck.ResultKeyAdvisory,
		)
		result.AddRecommendation("Configure a specific replica count (at least 3) for the ingress controller")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/networking/index#configuring-ingress", version))
		result.Detail = ingressPods
		return result, nil
	}

	// Parse the replica count
	replicas, parseErr := strconv.Atoi(replicaStr)
	if parseErr != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			fmt.Sprintf("Failed to parse ingress controller replica count: %s", replicaStr),
			healthcheck.ResultKeyAdvisory,
		), nil
	}

	// Check if it has at least 3 replicas
	if replicas < 3 {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			fmt.Sprintf("Ingress controller has insufficient replicas: %d (recommended: >= 3)", replicas),
			healthcheck.ResultKeyRecommended,
		)
		result.AddRecommendation("Increase the number of ingress controller replicas to at least 3 for high availability")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/networking/index#configuring-ingress", version))
		result.Detail = ingressPods
		return result, nil
	}

	// Has sufficient replicas
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		fmt.Sprintf("Ingress controller has sufficient replicas: %d", replicas),
		healthcheck.ResultKeyNoChange,
	)
	result.Detail = ingressPods
	return result, nil
}
