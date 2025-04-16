/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for log forwarding configuration. It:

- Checks if log forwarding is configured for long-term storage
- Examines cluster log forwarder configurations
- Identifies log forwarding targets and methods
- Provides recommendations for external log storage
- Helps ensure logs are preserved beyond the cluster's local storage

This check helps administrators ensure that application logs are properly forwarded to external systems for long-term retention and analysis.
*/

package monitoring

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// LoggingForwarderCheck checks if log forwarding is configured
type LoggingForwarderCheck struct {
	healthcheck.BaseCheck
}

// NewLoggingForwarderCheck creates a new log forwarder check
func NewLoggingForwarderCheck() *LoggingForwarderCheck {
	return &LoggingForwarderCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"logging-forwarder",
			"Log Forwarding",
			"Checks if log forwarding is configured for long-term storage",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *LoggingForwarderCheck) Run() (healthcheck.Result, error) {
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

	// Check if external forwarding is configured
	var detailedOut string
	var formattedDetailOut string
	var loggingTypeInfo string

	if loggingInfo.Type == LoggingTypeLoki {
		detailedOut, _ = utils.RunCommand("oc", "get", "clusterlogforwarders.observability.openshift.io", "-n", "openshift-logging", "-o", "yaml")
		loggingTypeInfo = "Logging Type: Loki-based logging\n\n"
	} else {
		detailedOut, _ = utils.RunCommand("oc", "get", "clusterlogforwarder", "-n", "openshift-logging", "-o", "yaml")
		loggingTypeInfo = "Logging Type: Traditional logging with Elasticsearch\n\n"
	}

	// Format the detailed output with proper AsciiDoc formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut = fmt.Sprintf("Log Forwarder Configuration:\n[source, yaml]\n----\n%s\n----\n\n", detailedOut)
	} else {
		formattedDetailOut = "Log Forwarder Configuration: No information available\n\n"
	}

	formattedDetailOut = loggingTypeInfo + formattedDetailOut

	// Check if we have external forwarding configured
	if !loggingInfo.HasExternalForwarder {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No external log forwarding configured",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Configure external log forwarding for long-term storage and better log management")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-external", version))

		result.Detail = formattedDetailOut
		return result, nil
	}

	// External forwarding is configured
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("%s with external forwarding is properly configured", string(loggingInfo.Type)),
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailOut
	return result, nil
}
