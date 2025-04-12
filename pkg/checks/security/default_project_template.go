package security

import (
	"fmt"
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
			healthcheck.CategorySecurity,
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
			healthcheck.StatusCritical,
			"Failed to check default project template",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking default project template: %v", err)
	}

	// Get detailed information using oc command for the report
	detailedOut, err := utils.RunCommand("oc", "get", "project.config.openshift.io/cluster", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed project configuration information"
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
			healthcheck.StatusWarning,
			"No default project template is configured",
			healthcheck.ResultKeyRecommended,
		)

		result.AddRecommendation("Configure a default project template to enforce consistent settings across new projects")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/building_applications/index#configuring-project-creation", version))
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/building_applications/index#quotas-setting-per-project", version))

		result.Detail = detailedOut
		return result, nil
	}

	// Default project template is configured
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		"Default project template is configured",
		healthcheck.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
