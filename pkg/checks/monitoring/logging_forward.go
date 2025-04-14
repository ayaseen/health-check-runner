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

	// Check for ClusterLogForwarder
	clusterForwarderOut, err := utils.RunCommand("oc", "get", "clusterlogforwarder", "instance", "-n", "openshift-logging", "-o", "yaml")

	clusterForwarderConfigured := err == nil

	// Get detailed information for the report (but redact any sensitive info)
	detailedOut := "ClusterLogForwarder configuration details not shown for security reasons"
	if clusterForwarderConfigured {
		detailedOut = clusterForwarderOut
	}

	// Check if operations logs are configured in the ClusterLogForwarder
	opsLogsConfigured := false
	if clusterForwarderConfigured {
		// Look for infrastructure or audit logs in the configuration
		opsLogsConfigured = strings.Contains(clusterForwarderOut, "infrastructure") ||
			strings.Contains(clusterForwarderOut, "audit")
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Check if there are any outputs configured
	outputsConfigured := false
	if clusterForwarderConfigured {
		outputsOut, err := utils.RunCommand("oc", "get", "clusterlogforwarder", "instance", "-n", "openshift-logging", "-o", "jsonpath={.spec.outputs}")
		outputsConfigured = err == nil && strings.TrimSpace(outputsOut) != "[]" && strings.TrimSpace(outputsOut) != ""
	}

	// Evaluate log forwarding configuration
	if !clusterForwarderConfigured {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"ClusterLogForwarder is not configured",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure ClusterLogForwarder to forward logs to external systems")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-external", version))
		result.Detail = "No ClusterLogForwarder configuration found"
		return result, nil
	}

	if !outputsConfigured {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"ClusterLogForwarder is configured but no outputs are defined",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure outputs in ClusterLogForwarder to forward logs to external systems")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-external", version))
		result.Detail = detailedOut
		return result, nil
	}

	if !opsLogsConfigured {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"ClusterLogForwarder is configured but operations logs are not included",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure forwarding for infrastructure and audit logs")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-collector-log-forwarding-about_cluster-logging-external", version))
		result.Detail = detailedOut
		return result, nil
	}

	// All checks passed
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Operations logs are properly configured for forwarding",
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
