package security

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// SelfProvisionerCheck checks if the self-provisioner role binding is configured securely
type SelfProvisionerCheck struct {
	healthcheck.BaseCheck
}

// NewSelfProvisionerCheck creates a new self-provisioner check
func NewSelfProvisionerCheck() *SelfProvisionerCheck {
	return &SelfProvisionerCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"self-provisioner",
			"Self Provisioner",
			"Checks if the self-provisioner role binding is configured to prevent uncontrolled namespace creation",
			healthcheck.CategorySecurity,
		),
	}
}

// Run executes the health check
func (c *SelfProvisionerCheck) Run() (healthcheck.Result, error) {
	// Get the self-provisioner cluster role binding
	out, err := utils.RunCommand("oc", "describe", "clusterrolebindings", "self-provisioners")
	if err != nil {
		// This might mean the role binding doesn't exist or has been renamed
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			"Self-provisioner role binding not found, which may indicate it has been removed or renamed",
			healthcheck.ResultKeyNoChange,
		), nil
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "clusterrolebindings", "self-provisioners", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed self-provisioners role binding"
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Check if the role binding includes system:authenticated:oauth
	if strings.Contains(out, "system:authenticated:oauth") {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"Self-provisioner role binding includes system:authenticated:oauth, allowing uncontrolled namespace creation",
			healthcheck.ResultKeyRecommended,
		)

		result.AddRecommendation("Remove the self-provisioner role from the system:authenticated:oauth group")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/building_applications/index#disabling-project-self-provisioning_configuring-project-creation", version))

		result.Detail = detailedOut
		return result, nil
	}

	// The self-provisioner role binding doesn't include system:authenticated:oauth
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		"Self-provisioner role binding is properly configured to prevent uncontrolled namespace creation",
		healthcheck.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
