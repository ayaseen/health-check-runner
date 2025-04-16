/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for OpenShift Logging component health. It:

- Examines the status of logging components based on the installed type
- Checks Elasticsearch or Loki pod status and health
- Verifies collector (Fluentd/Vector) functionality
- Identifies issues with logging component stability
- Provides recommendations for addressing logging health issues

This check ensures that the deployed logging solution is functioning properly and collecting logs reliably.
*/

package monitoring

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// LoggingHealthCheck checks if OpenShift Logging components are healthy
type LoggingHealthCheck struct {
	healthcheck.BaseCheck
}

// NewLoggingHealthCheck creates a new logging health check
func NewLoggingHealthCheck() *LoggingHealthCheck {
	return &LoggingHealthCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"logging-health",
			"OpenShift Logging Health",
			"Checks if OpenShift Logging components are functioning and healthy",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *LoggingHealthCheck) Run() (healthcheck.Result, error) {
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

	// Get the OpenShift version for recommendations
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.14" // Update to a more recent default version
	}

	// Check health based on the logging type
	var loggingPodsOut string
	var collectorPodsOut string
	var statusOutput string
	var unhealthyPods bool

	if loggingInfo.Type == LoggingTypeLoki {
		// For Loki, check pod status
		lokiPodsOut, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-logging", "-l", "app.kubernetes.io/component=loki")
		loggingPodsOut = lokiPodsOut

		if err == nil {
			// Check if any pods are not running
			unhealthyPods = strings.Contains(lokiPodsOut, "CrashLoopBackOff") ||
				strings.Contains(lokiPodsOut, "Error") ||
				strings.Contains(lokiPodsOut, "Failed")
		}

		// Check collector status
		collectorPodsOut, _ = utils.RunCommand("oc", "get", "pods", "-n", "openshift-logging", "-l", "app.kubernetes.io/component=collector")
	} else {
		// Traditional logging with ElasticSearch
		statusOutput, _ = utils.RunCommand("oc", "get", "clusterlogging", "instance", "-n", "openshift-logging", "-o", "yaml")

		// Check Elasticsearch status
		esPodsOut, _ := utils.RunCommand("oc", "get", "pods", "-n", "openshift-logging", "-l", "component=elasticsearch")
		loggingPodsOut = esPodsOut

		// Check if status is unhealthy
		unhealthyPods = strings.Contains(statusOutput, "status: yellow") ||
			strings.Contains(statusOutput, "status: red") ||
			strings.Contains(esPodsOut, "CrashLoopBackOff") ||
			strings.Contains(esPodsOut, "Error") ||
			strings.Contains(esPodsOut, "Failed")
	}

	// Format the detailed output with proper AsciiDoc formatting
	var formattedDetailOut strings.Builder

	// Add logging type information
	if loggingInfo.Type == LoggingTypeLoki {
		formattedDetailOut.WriteString("Logging Type: Loki-based logging\n\n")
	} else {
		formattedDetailOut.WriteString("Logging Type: Traditional logging with Elasticsearch\n\n")
	}

	// Add status output if available
	if strings.TrimSpace(statusOutput) != "" {
		formattedDetailOut.WriteString("Logging Status Information:\n[source, yaml]\n----\n")
		formattedDetailOut.WriteString(statusOutput)
		formattedDetailOut.WriteString("\n----\n\n")
	}

	// Add logging pods information
	if strings.TrimSpace(loggingPodsOut) != "" {
		formattedDetailOut.WriteString("Logging Pods Status:\n[source, bash]\n----\n")
		formattedDetailOut.WriteString(loggingPodsOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Logging Pods Status: No information available\n\n")
	}

	// Add collector pods information if available
	if strings.TrimSpace(collectorPodsOut) != "" {
		formattedDetailOut.WriteString("Collector Pods Status:\n[source, bash]\n----\n")
		formattedDetailOut.WriteString(collectorPodsOut)
		formattedDetailOut.WriteString("\n----\n\n")
	}

	// Check if there are unhealthy pods or components
	if unhealthyPods {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("%s Logging is not healthy (status is degraded)", string(loggingInfo.Type)),
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Investigate the root cause of logging issues")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-cluster-status", version))

		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Logging is healthy
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("%s Logging is healthy", string(loggingInfo.Type)),
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailOut.String()
	return result, nil
}
