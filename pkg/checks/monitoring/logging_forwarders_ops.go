/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for operations log forwarding. It:

- Verifies if infrastructure and audit logs are being forwarded
- Examines cluster log forwarder configurations
- Checks for proper forwarding of operations logs to external systems
- Provides recommendations for comprehensive log management
- Helps ensure important system logs are preserved and analyzed

This check ensures that critical operations logs are properly captured and preserved for troubleshooting and compliance.
*/

package monitoring

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// LoggingForwardersOpsCheck checks if OpenShift logging is configured to forward operations logs
type LoggingForwardersOpsCheck struct {
	healthcheck.BaseCheck
}

// NewLoggingForwardersOpsCheck creates a new logging forwarders OPS check
func NewLoggingForwardersOpsCheck() *LoggingForwardersOpsCheck {
	return &LoggingForwardersOpsCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"logging-forwarders-ops",
			"Logging Forwarders OPS",
			"Checks if OpenShift logging is configured to forward operations logs",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *LoggingForwardersOpsCheck) Run() (healthcheck.Result, error) {
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

	// Check if operations logs are forwarded
	var opsLogsForwarded bool
	var detailedOut string
	var formattedDetailOut string

	if loggingInfo.Type == LoggingTypeLoki {
		clfoOut, err := utils.RunCommand("oc", "get", "clusterlogforwarders.observability.openshift.io", "-n", "openshift-logging", "-o", "yaml")
		if err == nil {
			detailedOut = clfoOut
			// Check if infrastructure or audit logs are included in pipelines
			opsLogsForwarded = strings.Contains(clfoOut, "infrastructure") || strings.Contains(clfoOut, "audit")
		}
	} else {
		// Traditional logging with ElasticSearch
		clfoOut, err := utils.RunCommand("oc", "get", "clusterlogforwarder", "-n", "openshift-logging", "-o", "yaml")
		if err == nil {
			detailedOut = clfoOut
			// Check if infrastructure or audit logs are included in pipelines
			opsLogsForwarded = strings.Contains(clfoOut, "infrastructure") || strings.Contains(clfoOut, "audit")
		}
	}

	// Format the detailed output with proper AsciiDoc formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut = fmt.Sprintf("Log Forwarder Configuration:\n[source, yaml]\n----\n%s\n----\n\n", detailedOut)
	} else {
		formattedDetailOut = "Log Forwarder Configuration: No information available\n\n"
	}

	// Add logging type information
	var loggingTypeInfo string
	if loggingInfo.Type == LoggingTypeLoki {
		loggingTypeInfo = "Logging Type: Loki-based logging\n\n"
	} else {
		loggingTypeInfo = "Logging Type: Traditional logging with Elasticsearch\n\n"
	}

	formattedDetailOut = loggingTypeInfo + formattedDetailOut

	// Check if external forwarding is configured, but operations logs are not included
	if loggingInfo.HasExternalForwarder && !opsLogsForwarded {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Log forwarding is configured but operations logs are not included",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Configure forwarding for infrastructure and audit logs")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-collector-log-forwarding-about_cluster-logging-external", version))

		result.Detail = formattedDetailOut
		return result, nil
	} else if loggingInfo.HasExternalForwarder && opsLogsForwarded {
		// External forwarding with operations logs is properly configured
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"Operations logs are properly configured for forwarding",
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailOut
		return result, nil
	} else {
		// No external forwarding is configured
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No external log forwarding is configured for operations logs",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Configure external forwarding for infrastructure and audit logs")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-external", version))

		result.Detail = formattedDetailOut
		return result, nil
	}
}
