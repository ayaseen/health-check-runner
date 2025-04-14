package monitoring

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
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
	out, err := utils.RunCommand("oc", "get", "clusterlogforwarder", "-n", "openshift-logging", "-o", "yaml")
	if err != nil {
		// ClusterLogForwarder might not exist
		return healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Log forwarding is not configured",
			types.ResultKeyRecommended,
		), nil
	}

	// Check if it's properly configured with outputs, pipelines, etc.
	emptyConfig := strings.Contains(out, "outputs: []") || strings.Contains(out, "pipelines: []") || strings.Contains(out, "inputs: []")

	// Get the OpenShift version for recommendations
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Fallback version
	}

	if emptyConfig {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Log forwarding is configured but has empty pipelines or outputs",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Configure proper log forwarding for long-term storage")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/logging/index#cluster-logging-external", version))

		result.Detail = out
		return result, nil
	}

	// Log forwarding is properly configured
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Log forwarding is properly configured",
		types.ResultKeyNoChange,
	)
	result.Detail = out
	return result, nil
}
