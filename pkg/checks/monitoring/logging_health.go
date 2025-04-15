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
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
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
	if loggingInfo.Type == LoggingTypeLoki {
		// For Loki, check pod status
		lokiPodsOut, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-logging", "-l", "app.kubernetes.io/component=loki")
		if err != nil {
			return healthcheck.NewResult(
				c.ID(),
				types.StatusCritical,
				"Failed to get Loki pod status",
				types.ResultKeyRequired,
			), fmt.Errorf("error getting Loki pod status: %v", err)
		}

		// Check if any pods are not running
		if strings.Contains(lokiPodsOut, "CrashLoopBackOff") ||
			strings.Contains(lokiPodsOut, "Error") ||
			strings.Contains(lokiPodsOut, "Failed") {

			result := healthcheck.NewResult(
				c.ID(),
				types.StatusWarning,
				"Loki Logging has unhealthy pods",
				types.ResultKeyRecommended,
			)

			result.AddRecommendation("Investigate the pod issues in the openshift-logging namespace")
			result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index", version))

			result.Detail = lokiPodsOut
			return result, nil
		}

		// Check collector status
		collectorPodsOut, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-logging", "-l", "app.kubernetes.io/component=collector")
		if err == nil {
			if strings.Contains(collectorPodsOut, "CrashLoopBackOff") ||
				strings.Contains(collectorPodsOut, "Error") ||
				strings.Contains(collectorPodsOut, "Failed") {

				result := healthcheck.NewResult(
					c.ID(),
					types.StatusWarning,
					"Logging collector pods are unhealthy",
					types.ResultKeyRecommended,
				)

				result.AddRecommendation("Investigate the collector pod issues in the openshift-logging namespace")
				result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index", version))

				result.Detail = fmt.Sprintf("Loki pods:\n%s\n\nCollector pods:\n%s", lokiPodsOut, collectorPodsOut)
				return result, nil
			}
		}

		// Logging is healthy
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"Loki Logging is healthy",
			types.ResultKeyNoChange,
		)
		result.Detail = lokiPodsOut
		return result, nil
	} else {
		// For traditional logging, check the ClusterLogging status
		clStatus, err := utils.RunCommand("oc", "get", "clusterlogging", "instance", "-n", "openshift-logging", "-o", "yaml")
		if err != nil {
			return healthcheck.NewResult(
				c.ID(),
				types.StatusCritical,
				"Failed to get traditional logging status",
				types.ResultKeyRequired,
			), fmt.Errorf("error getting traditional logging status: %v", err)
		}

		// Check Elasticsearch status
		esPodsOut, _ := utils.RunCommand("oc", "get", "pods", "-n", "openshift-logging", "-l", "component=elasticsearch")

		// Check if status is unhealthy
		isUnhealthy := strings.Contains(clStatus, "status: yellow") ||
			strings.Contains(clStatus, "status: red") ||
			strings.Contains(esPodsOut, "CrashLoopBackOff") ||
			strings.Contains(esPodsOut, "Error") ||
			strings.Contains(esPodsOut, "Failed")

		if isUnhealthy {
			result := healthcheck.NewResult(
				c.ID(),
				types.StatusWarning,
				"Traditional Logging is not healthy (status is degraded)",
				types.ResultKeyRecommended,
			)

			result.AddRecommendation("Investigate the root cause of logging issues")
			result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-cluster-status", version))

			result.Detail = fmt.Sprintf("Cluster logging status:\n%s\n\nElasticsearch pods:\n%s", clStatus, esPodsOut)
			return result, nil
		}

		// Logging is healthy
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"Traditional Logging is healthy",
			types.ResultKeyNoChange,
		)
		result.Detail = clStatus
		return result, nil
	}
}
