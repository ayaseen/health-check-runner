/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for OpenShift Logging installation. It:

- Detects if OpenShift Logging is installed and configured
- Identifies the logging stack type (Elasticsearch or Loki)
- Examines logging operator configurations
- Provides recommendations for deploying logging if not installed
- Helps ensure proper log aggregation capabilities

This check verifies the presence and basic configuration of the logging subsystem in OpenShift.
*/

package monitoring

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
	"strings"
)

// LoggingInstallCheck checks if OpenShift Logging is installed and configured
type LoggingInstallCheck struct {
	healthcheck.BaseCheck
}

// NewLoggingInstallCheck creates a new logging check
func NewLoggingInstallCheck() *LoggingInstallCheck {
	return &LoggingInstallCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"logging-install",
			"OpenShift Logging Installation",
			"Checks if OpenShift Logging is installed and configured correctly",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *LoggingInstallCheck) Run() (healthcheck.Result, error) {
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

	// Get detailed information for the report
	var clfoOut, lokiOut string
	if loggingInfo.Type == LoggingTypeLoki {
		// Get cluster log forwarder info
		tempClfoOut, detailErr := utils.RunCommand("oc", "get", "clusterlogforwarders.observability.openshift.io", "-n", "openshift-logging", "-o", "yaml")
		if detailErr == nil {
			clfoOut = tempClfoOut
		} else {
			clfoOut = "Failed to get detailed Loki logging information"
		}

		// Add Loki Stack info
		tempLokiOut, lokiErr := utils.RunCommand("oc", "get", "lokistack", "-n", "openshift-logging", "-o", "yaml")
		if lokiErr == nil {
			lokiOut = tempLokiOut
		}
	} else if loggingInfo.Type == LoggingTypeTraditional {
		clOut, detailErr := utils.RunCommand("oc", "get", "clusterlogging", "-n", "openshift-logging", "-o", "yaml")
		if detailErr == nil {
			clfoOut = clOut
		} else {
			clfoOut = "Failed to get detailed traditional logging information"
		}
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailedOut string

	// Format cluster log forwarder output
	if strings.TrimSpace(clfoOut) != "" {
		if loggingInfo.Type == LoggingTypeLoki {
			formattedDetailedOut = fmt.Sprintf("Cluster Log Forwarder Configuration:\n[source, yaml]\n----\n%s\n----\n\n", clfoOut)
		} else {
			formattedDetailedOut = fmt.Sprintf("Cluster Logging Configuration:\n[source, yaml]\n----\n%s\n----\n\n", clfoOut)
		}
	} else {
		formattedDetailedOut = "Logging Configuration: No information available\n\n"
	}

	// Format Loki Stack output separately
	if strings.TrimSpace(lokiOut) != "" {
		formattedDetailedOut += fmt.Sprintf("Loki Stack Configuration:\n[source, yaml]\n----\n%s\n----\n\n", lokiOut)
	}

	// Get the OpenShift version for recommendations
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.14" // Update to a more recent default version
	}

	// Generate result based on the detected configuration
	if loggingInfo.Type == LoggingTypeNone {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"OpenShift Logging is not installed",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Deploy the logging subsystem to aggregate logs from your OpenShift Container Platform cluster")
		result.AddRecommendation(fmt.Sprintf("Follow the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/cluster-logging-deploying", version))

		result.Detail = formattedDetailedOut
		return result, nil
	} else if loggingInfo.Type == LoggingTypeLoki {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"OpenShift Logging with Loki is installed and configured",
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailedOut
		return result, nil
	} else {
		// Traditional logging
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"OpenShift Logging with Elasticsearch is installed and configured",
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailedOut
		return result, nil
	}
}
