package monitoring

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
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
	var detailedOut string
	if loggingInfo.Type == LoggingTypeLoki {
		clfoOut, detailErr := utils.RunCommand("oc", "get", "clusterlogforwarders.observability.openshift.io", "-n", "openshift-logging", "-o", "yaml")
		if detailErr == nil {
			detailedOut = clfoOut
		} else {
			detailedOut = "Failed to get detailed Loki logging information"
		}

		// Add Loki Stack info
		lokiOut, lokiErr := utils.RunCommand("oc", "get", "lokistack", "-n", "openshift-logging", "-o", "yaml")
		if lokiErr == nil {
			detailedOut += "\n\n=== Loki Stack Configuration ===\n" + lokiOut
		}
	} else if loggingInfo.Type == LoggingTypeTraditional {
		clOut, detailErr := utils.RunCommand("oc", "get", "clusterlogging", "-n", "openshift-logging", "-o", "yaml")
		if detailErr == nil {
			detailedOut = clOut
		} else {
			detailedOut = "Failed to get detailed traditional logging information"
		}
	} else {
		detailedOut = "No logging configuration found"
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

		result.Detail = detailedOut
		return result, nil
	} else if loggingInfo.Type == LoggingTypeLoki {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"OpenShift Logging with Loki is installed and configured",
			types.ResultKeyNoChange,
		)
		result.Detail = detailedOut
		return result, nil
	} else {
		// Traditional logging
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"OpenShift Logging with Elasticsearch is installed and configured",
			types.ResultKeyNoChange,
		)
		result.Detail = detailedOut
		return result, nil
	}
}
