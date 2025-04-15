/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for alert forwarding configuration. It:

- Checks if OpenShift alerts are forwarded to external notification systems
- Examines Alertmanager configuration for receivers and routes
- Identifies integrations with systems like PagerDuty, Slack, or email
- Provides recommendations for alert notification setup
- Helps ensure timely response to critical cluster events

This check helps administrators ensure that alerts are properly routed to notification systems for timely response to issues.
*/

package monitoring

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// AlertsForwardingCheck checks if OpenShift alerts are forwarded to an external system
type AlertsForwardingCheck struct {
	healthcheck.BaseCheck
}

// NewAlertsForwardingCheck creates a new alerts forwarding check
func NewAlertsForwardingCheck() *AlertsForwardingCheck {
	return &AlertsForwardingCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"alerts-forwarding",
			"Alerts Forwarding",
			"Checks if OpenShift alerts are forwarded to an external system",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *AlertsForwardingCheck) Run() (healthcheck.Result, error) {
	// Check if Alertmanager is configured for forwarding
	out, err := utils.RunCommand("oc", "get", "secret", "alertmanager-main", "-n", "openshift-monitoring", "-o", "jsonpath={.data.alertmanager\\.yaml}", "|", "base64", "-d", "|", "grep", "-E", "receiver|webhook_configs|pagerduty_configs|email_configs|slack_configs")

	var alertForwardingConfigured bool
	if err == nil && strings.TrimSpace(out) != "" {
		alertForwardingConfigured = true
	}

	// Check for specific integrations that indicate external forwarding
	integrationsOut, err := utils.RunCommand("oc", "get", "secret", "alertmanager-main", "-n", "openshift-monitoring", "-o", "jsonpath={.data.alertmanager\\.yaml}", "|", "base64", "-d")

	var externalSystemsConfigured bool
	if err == nil && strings.TrimSpace(integrationsOut) != "" {
		// Check for common external integration keywords
		externalSystemsConfigured = strings.Contains(strings.ToLower(integrationsOut), "pagerduty") ||
			strings.Contains(strings.ToLower(integrationsOut), "slack") ||
			strings.Contains(strings.ToLower(integrationsOut), "webhook") ||
			strings.Contains(strings.ToLower(integrationsOut), "victorops") ||
			strings.Contains(strings.ToLower(integrationsOut), "pushover") ||
			strings.Contains(strings.ToLower(integrationsOut), "opsgenie") ||
			strings.Contains(strings.ToLower(integrationsOut), "wechat")
	}

	// Get detailed information for the report (redact any sensitive info)
	detailedOut := "Alertmanager configuration details not shown for security reasons"

	// Check if monitoring is configured with remote-write
	remoteWriteOut, err := utils.RunCommand("oc", "get", "prometheuses.monitoring.coreos.com", "-n", "openshift-monitoring", "k8s", "-o", "yaml", "|", "grep", "-A", "5", "remoteWrite")
	remoteWriteConfigured := err == nil && strings.Contains(remoteWriteOut, "remoteWrite")

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Evaluate alert forwarding configuration
	if !alertForwardingConfigured {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Alertmanager is not configured to forward alerts",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure Alertmanager to forward alerts to an external notification system")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/managing-alerts", version))
		result.Detail = detailedOut
		return result, nil
	}

	if !externalSystemsConfigured {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Alertmanager is configured but no external notification systems detected",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure integration with external notification systems like PagerDuty, Slack, or email")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/managing-alerts", version))
		result.Detail = detailedOut
		return result, nil
	}

	// If we also have remote-write configured, that's even better
	if remoteWriteConfigured {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"Alerts are forwarded to external systems and metrics are remote-written",
			types.ResultKeyNoChange,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// Standard case - external forwarding configured
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Alerts are forwarded to external notification systems",
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
