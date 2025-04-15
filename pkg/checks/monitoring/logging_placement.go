/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for logging component placement. It:

- Verifies if logging components are scheduled on infrastructure nodes
- Checks node selector and affinity configurations for logging pods
- Examines pod placement relative to node roles
- Provides recommendations for optimal logging component placement
- Helps ensure proper resource allocation for logging workloads

This check helps maintain a properly architected environment where logging components are appropriately placed on dedicated infrastructure nodes.
*/

package monitoring

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// LoggingPlacementCheck checks if logging components are placed on appropriate nodes
type LoggingPlacementCheck struct {
	healthcheck.BaseCheck
}

// NewLoggingPlacementCheck creates a new logging placement check
func NewLoggingPlacementCheck() *LoggingPlacementCheck {
	return &LoggingPlacementCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"logging-placement",
			"Logging Component Placement",
			"Checks if logging components are scheduled on appropriate nodes",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *LoggingPlacementCheck) Run() (healthcheck.Result, error) {
	// Detect logging configuration
	loggingInfo, err := DetectLoggingConfiguration()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to detect logging configuration",
			types.ResultKeyRequired,
		), fmt.Errorf("error detecting logging configuration: %v", err)
	}

	// If logging is not installed, return NotApplicable
	if loggingInfo.Type == LoggingTypeNone {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"OpenShift Logging is not installed",
			types.ResultKeyNotApplicable,
		), nil
	}

	// Check if infrastructure nodes exist
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

	// Get infrastructure node names
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

	// Check pod placement based on logging type
	var allOnInfraNodes bool
	var podsOnNonInfraNodes []string
	var podOut string

	if loggingInfo.Type == LoggingTypeLoki {
		// Check Loki pods
		podOut, err = utils.RunCommand("oc", "get", "pods", "-n", "openshift-logging", "-l", "app.kubernetes.io/component=loki", "-o", "wide")
		if err != nil {
			return healthcheck.NewResult(
				c.ID(),
				types.StatusCritical,
				"Failed to get Loki pods",
				types.ResultKeyRequired,
			), fmt.Errorf("error getting Loki pods: %v", err)
		}
	} else {
		// Check Elasticsearch pods
		podOut, err = utils.RunCommand("oc", "get", "pods", "-n", "openshift-logging", "-l", "component=elasticsearch", "-o", "wide")
		if err != nil {
			return healthcheck.NewResult(
				c.ID(),
				types.StatusCritical,
				"Failed to get Elasticsearch pods",
				types.ResultKeyRequired,
			), fmt.Errorf("error getting Elasticsearch pods: %v", err)
		}
	}

	// Check if all pods are on infrastructure nodes
	allOnInfraNodes = true
	for _, line := range strings.Split(podOut, "\n") {
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
		version = "4.14" // Update to a more recent default version
	}

	if !allOnInfraNodes {
		var componentName string
		if loggingInfo.Type == LoggingTypeLoki {
			componentName = "Loki"
		} else {
			componentName = "Elasticsearch"
		}

		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Some %s pods are not scheduled on infrastructure nodes", componentName),
			types.ResultKeyRecommended,
		)

		result.AddRecommendation(fmt.Sprintf("Move %s pods to infrastructure nodes", componentName))
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#infrastructure-moving-logging_cluster-logging-moving", version))

		detail := fmt.Sprintf("%s pods not on infrastructure nodes:\n%s\n\nInfrastructure nodes:\n%s\n\nPod details:\n%s",
			componentName,
			strings.Join(podsOnNonInfraNodes, "\n"),
			nodeOut,
			podOut)
		result.Detail = detail
		return result, nil
	}

	// All pods are on infrastructure nodes
	var componentName string
	if loggingInfo.Type == LoggingTypeLoki {
		componentName = "Loki"
	} else {
		componentName = "Elasticsearch"
	}

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("All %s pods are scheduled on infrastructure nodes", componentName),
		types.ResultKeyNoChange,
	)
	result.Detail = podOut
	return result, nil
}
