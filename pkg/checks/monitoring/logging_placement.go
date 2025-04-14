package monitoring

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// LoggingPlacementCheck checks if Elasticsearch pods are placed on appropriate nodes
type LoggingPlacementCheck struct {
	healthcheck.BaseCheck
}

// NewLoggingPlacementCheck creates a new logging placement check
func NewLoggingPlacementCheck() *LoggingPlacementCheck {
	return &LoggingPlacementCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"logging-placement",
			"Logging Component Placement",
			"Checks if Elasticsearch pods are scheduled on appropriate nodes",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *LoggingPlacementCheck) Run() (healthcheck.Result, error) {
	// First check if logging is installed
	isInstalled, err := isLoggingInstalled()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to check OpenShift Logging installation",
			types.ResultKeyRequired,
		), fmt.Errorf("error checking logging installation: %v", err)
	}

	// If logging is not installed, return NotApplicable
	if !isInstalled {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"OpenShift Logging is not installed",
			types.ResultKeyNotApplicable,
		), nil
	}

	// Get Elasticsearch pods
	out, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-logging", "-l", "component=elasticsearch", "-o", "wide")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get Elasticsearch pods",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting Elasticsearch pods: %v", err)
	}

	// Get node information
	nodeOut, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=", "-o", "name")
	if err != nil || nodeOut == "" {
		// There are no infra nodes defined, so this check is not applicable
		return healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No infrastructure nodes found in the cluster",
			types.ResultKeyRecommended,
		), nil
	}

	// Check if all pods are on infra nodes
	infraNodeNames := []string{}
	for _, line := range strings.Split(nodeOut, "\n") {
		if line != "" {
			// Extract node name from format like "node/node-name"
			parts := strings.Split(line, "/")
			if len(parts) > 1 {
				infraNodeNames = append(infraNodeNames, parts[1])
			}
		}
	}

	allOnInfraNodes := true
	podsOnNonInfraNodes := []string{}

	for _, line := range strings.Split(out, "\n") {
		if line != "" && !strings.HasPrefix(line, "NAME") { // Skip header
			fields := strings.Fields(line)
			if len(fields) >= 7 { // Ensure we have enough fields to access node name
				podName := fields[0]
				nodeName := fields[6]

				onInfraNode := false
				for _, infraNode := range infraNodeNames {
					if nodeName == infraNode {
						onInfraNode = true
						break
					}
				}

				if !onInfraNode {
					allOnInfraNodes = false
					podsOnNonInfraNodes = append(podsOnNonInfraNodes, fmt.Sprintf("%s on %s", podName, nodeName))
				}
			}
		}
	}

	// Get the OpenShift version for recommendations
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Fallback version
	}

	if !allOnInfraNodes {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Some Elasticsearch pods are not scheduled on infrastructure nodes",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Move Elasticsearch pods to infrastructure nodes")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#infrastructure-moving-logging_cluster-logging-moving", version))

		detail := fmt.Sprintf("Elasticsearch pods not on infrastructure nodes:\n%s\n\nInfrastructure nodes:\n%s\n\nPod details:\n%s",
			strings.Join(podsOnNonInfraNodes, "\n"),
			nodeOut,
			out)
		result.Detail = detail
		return result, nil
	}

	// All pods are on infra nodes
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"All Elasticsearch pods are scheduled on infrastructure nodes",
		types.ResultKeyNoChange,
	)
	result.Detail = out
	return result, nil
}
