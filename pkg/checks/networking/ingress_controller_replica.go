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
	minRecommendedReplicas int
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
		minRecommendedReplicas: 3, // Minimum recommended for production
	}
}

// Run executes the health check
func (c *IngressControllerReplicaCheck) Run() (healthcheck.Result, error) {
	// Get the ingress controller replica count
	out, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator", "-o", "jsonpath={.spec.replicas}")

	// Parse replica count
	var replicaCount int
	replicaCount = 0 // Default if not specified (auto-scaled based on the infrastructure)

	if err == nil && strings.TrimSpace(out) != "" {
		replicaCount, err = strconv.Atoi(strings.TrimSpace(out))
		if err != nil {
			replicaCount = 0
		}
	}

	// If replica count is not specified, check the actual router deployment
	if replicaCount == 0 {
		deploymentOut, err := utils.RunCommand("oc", "get", "deployment", "-n", "openshift-ingress", "-l", "ingresscontroller.operator.openshift.io/deployment-ingresscontroller=default", "-o", "jsonpath={.items[0].spec.replicas}")
		if err == nil && strings.TrimSpace(deploymentOut) != "" {
			replicaCount, err = strconv.Atoi(strings.TrimSpace(deploymentOut))
			if err != nil {
				replicaCount = 0
			}
		}
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator", "-o", "yaml")
	if err != nil {
		detailedOut = "Failed to get detailed ingress controller configuration"
	}

	// Get the actual running router pod count
	podOut, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-ingress", "-l", "ingresscontroller.operator.openshift.io/deployment-ingresscontroller=default")
	runningPodCount := 0

	if err == nil {
		// Count lines excluding header
		lines := strings.Split(podOut, "\n")
		if len(lines) > 1 {
			runningPodCount = len(lines) - 1
			for i := 1; i < len(lines); i++ {
				if strings.TrimSpace(lines[i]) == "" {
					runningPodCount--
				}
			}
		}
	}

	// Get the node count to determine minimum replica recommendation
	nodeOut, err := utils.RunCommand("oc", "get", "nodes", "--no-headers")
	nodeCount := 0

	if err == nil {
		// Count non-empty lines
		lines := strings.Split(nodeOut, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				nodeCount++
			}
		}
	}

	// Adjust minimum recommendation based on node count
	// For very small clusters (3 nodes or less), 2 replicas might be acceptable
	minReplicas := c.minRecommendedReplicas
	if nodeCount > 0 && nodeCount <= 3 {
		minReplicas = 2
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Evaluate the replica count
	if replicaCount < minReplicas && runningPodCount < minReplicas {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Ingress controller has insufficient replicas (%d), recommend at least %d for high availability",
				replicaCount > 0 ? replicaCount : runningPodCount, minReplicas),
		types.ResultKeyRecommended,
	)
		result.AddRecommendation(fmt.Sprintf("Increase the number of ingress controller replicas to at least %d for production workloads", minReplicas))
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/networking/index#nw-ingress-controller-configuration_configuring-ingress", version))
		result.Detail = detailedOut
		return result, nil
	}

	// All checks passed
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("Ingress controller has sufficient replicas (%d)",
			replicaCount > 0 ? replicaCount : runningPodCount),
	types.ResultKeyNoChange,
)
	result.Detail = detailedOut
	return result, nil
}