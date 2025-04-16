/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements a health check for default project templates. It:

- Checks if a custom default project template is configured
- Verifies the project request template settings
- Provides recommendations for standardizing project creation
- Helps enforce consistent project settings across the cluster
- Ensures proper governance for new namespace creation

This check aids in maintaining consistent resource constraints and security settings across all newly created projects.
*/

package security

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// DefaultProjectTemplateCheck checks if a custom default project template is configured
type DefaultProjectTemplateCheck struct {
	healthcheck.BaseCheck
}

// NewDefaultProjectTemplateCheck creates a new default project template check
func NewDefaultProjectTemplateCheck() *DefaultProjectTemplateCheck {
	return &DefaultProjectTemplateCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"default-project-template",
			"Default Project Template",
			"Checks if a custom default project template is configured",
			types.CategorySecurity,
		),
	}
}

// Run executes the health check
func (c *DefaultProjectTemplateCheck) Run() (healthcheck.Result, error) {
	// Check if a custom default project template is configured
	out, err := utils.RunCommand("oc", "get", "project.config.openshift.io/cluster", "-o", "jsonpath={.spec.projectRequestTemplate}")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to check default project template",
			types.ResultKeyRequired,
		), fmt.Errorf("error checking default project template: %v", err)
	}

	// Get detailed information using oc command for the report
	detailedOut, err := utils.RunCommand("oc", "get", "project.config.openshift.io/cluster", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed project configuration information"
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Default Project Template Analysis ===\n\n")

	// Add project configuration with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut.WriteString("Project Configuration:\n[source, yaml]\n----\n")
		formattedDetailOut.WriteString(detailedOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Project Configuration: No information available\n\n")
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// If no default project template is configured
	if strings.TrimSpace(out) == "" {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No default project template is configured",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Configure a default project template to enforce consistent settings across new projects")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/building_applications/index#configuring-project-creation", version))
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/building_applications/index#quotas-setting-per-project", version))

		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Default project template is configured
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Default project template is configured",
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailOut.String()
	return result, nil
}
