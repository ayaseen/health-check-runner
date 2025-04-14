package networking

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// IngressControllerPlacementCheck checks if the ingress controller is properly placed on infrastructure nodes
type IngressControllerPlacementCheck struct {
	healthcheck.BaseCheck
}

// NewIngressControllerPlacementCheck creates a new ingress controller placement check
func NewIngressControllerPlacementCheck() *IngressControllerPlacementCheck {
	return &IngressControllerPlacementCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"ingress-controller-placement",
			"Ingress Controller Placement",
			"Checks if the ingress controller is properly placed on infrastructure nodes",
			types.CategoryNetworking,
		),
	}
}

// Run executes the health check
func (c *IngressControllerPlacementCheck) Run() (healthcheck.Result, error) {
	// Get the ingress controller configuration
	out, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator", "-o", "jsonpath={.spec.nodePlacement}")

	// Check if node placement is configured
	nodePlacementConfigured := err == nil && strings.TrimSpace(out) != "" && strings.TrimSpace(out) != "{}"

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator", "-o", "yaml")
	if err != nil {
		detailedOut = "Failed to get detailed ingress controller configuration"
	}

	// Check if it's specifically placed on infra nodes
	infraPlacementConfig, err := utils.RunCommand("oc", "get", "ingresscontroller/default", "-n", "openshift-ingress-operator", "-o", "jsonpath={.spec.nodePlacement.nodeSelector.matchLabels}")
	placedOnInfraNodes := err == nil && strings.Contains(infraPlacementConfig, "node-role.kubernetes.io/infra")

	// Check if the router pods are actually running on infra nodes
	routerPodsOut, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-ingress", "-l", "ingresscontroller.operator.openshift.io/deployment-ingresscontroller=default", "-o", "wide")

	// Check if infra nodes exist in the cluster
	infraNodesOut, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=")
	infraNodesExist := err == nil && !strings.Contains(infraNodesOut, "No resources found")

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// If there are no infra nodes, recommend creating them
	if !infraNodesExist {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No infrastructure nodes found for ingress controller placement",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Create dedicated infrastructure nodes for router and other infrastructure components")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/machine_management/index#creating-infrastructure-machinesets", version))
		result.Detail = detailedOut
		return result, nil
	}

	// If node placement isn't configured
	if !nodePlacementConfigured {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Ingress controller node placement is not configured",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure the ingress controller to be placed on infrastructure nodes")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/networking/index#nw-ingress-controller-configuration-parameters_configuring-ingress", version))
		result.Detail = detailedOut
		return result, nil
	}

	// If not placed on infra nodes specifically
	if !placedOnInfraNodes {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Ingress controller is not configured to be placed on infrastructure nodes",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure the ingress controller to be placed on infrastructure nodes using the node-role.kubernetes.io/infra label")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/networking/index#nw-ingress-controller-configuration-parameters_configuring-ingress", version))
		result.Detail = detailedOut
		return result, nil
	}

	// Check if the pods are actually running on infra nodes
	allPodsOnInfraNodes := true
	if strings.Contains(routerPodsOut, "NAME") {
		// Get list of infra node names
		infraNodeNames, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=", "-o", "jsonpath={.items[*].metadata.name}")
		if err == nil {
			infraNodesList := strings.Split(infraNodeNames, " ")
			infraNodesMap := make(map[string]bool)
			for _, node := range infraNodesList {
				if node != "" {
					infraNodesMap[node] = true
				}
			}

			// Check each router pod's node
			for _, line := range strings.Split(routerPodsOut, "\n")[1:] {
				if line == "" {
					continue
				}
				fields := strings.Fields(line)
				if len(fields) >= 7 {
					nodeName := fields[6]
					if !infraNodesMap[nodeName] {
						allPodsOnInfraNodes = false
						break
					}
				}
			}
		}
	}

	if !allPodsOnInfraNodes {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Some ingress controller pods are not running on infrastructure nodes",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Verify the node selector configuration and that all router pods are placed on infrastructure nodes")
		result.AddRecommendation("Check if the nodes have the correct labels and if there are any taints preventing scheduling")
		result.Detail = fmt.Sprintf("Pods status:\n%s\n\nIngress controller configuration:\n%s", routerPodsOut, detailedOut)
		return result, nil
	}

	// All checks passed
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Ingress controller is properly placed on infrastructure nodes",
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
