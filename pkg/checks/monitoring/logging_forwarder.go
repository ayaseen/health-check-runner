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
	if !loggingInfo.HasExternalForwarder {
		var detailedOut string
		var message string

		if loggingInfo.Type == LoggingTypeLoki {
			detailedOut, _ = utils.RunCommand("oc", "get", "clusterlogforwarders.observability.openshift.io", "-n", "openshift-logging", "-o", "yaml")
			message = "Loki logging is configured but no external forwarding is set up"
		} else {
			detailedOut, _ = utils.RunCommand("oc", "get", "clusterlogging", "-n", "openshift-logging", "-o", "yaml")
			message = "Traditional logging is configured but no external forwarding is set up"
		}

		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			message,
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Configure external log forwarding for long-term storage and better log management")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-external", version))

		result.Detail = detailedOut
		return result, nil
	}

	// External forwarding is configured
	var detailedOut string
	var message string

	if loggingInfo.Type == LoggingTypeLoki {
		detailedOut, _ = utils.RunCommand("oc", "get", "clusterlogforwarders.observability.openshift.io", "-n", "openshift-logging", "-o", "yaml")
		message = "Loki logging with external forwarding is properly configured"
	} else {
		detailedOut, _ = utils.RunCommand("oc", "get", "clusterlogforwarder", "-n", "openshift-logging", "-o", "yaml")
		message = "Traditional logging with external forwarding is properly configured"
	}

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		message,
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
