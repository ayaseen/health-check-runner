/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements a health check for the self-provisioner role binding. It:

- Checks if the self-provisioner role is configured securely
- Verifies that system:authenticated:oauth group doesn't have uncontrolled namespace creation rights
- Provides recommendations for proper project creation control
- Helps maintain proper multi-tenancy boundaries
- Prevents unauthorized project proliferation

This check helps maintain proper governance over namespace creation in shared OpenShift environments.
*/

package security

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
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
			types.CategorySecurity,
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
			types.StatusOK,
			"Self-provisioner role binding not found, which may indicate it has been removed or renamed",
			types.ResultKeyNoChange,
		), nil
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "clusterrolebindings", "self-provisioners", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed self-provisioners role binding"
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Self-Provisioner Role Binding Analysis ===\n\n")

	// Add role binding configuration with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut.WriteString("Self-Provisioners Role Binding:\n[source, yaml]\n----\n")
		formattedDetailOut.WriteString(detailedOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Self-Provisioners Role Binding: No information available\n\n")
	}

	// Add self-provisioner role binding description with proper formatting
	if strings.TrimSpace(out) != "" {
		formattedDetailOut.WriteString("Self-Provisioners Role Binding Description:\n[source, bash]\n----\n")
		formattedDetailOut.WriteString(out)
		formattedDetailOut.WriteString("\n----\n\n")
	}

	// Add security analysis
	formattedDetailOut.WriteString("=== Security Analysis ===\n\n")

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Check if the role binding includes system:authenticated:oauth
	if strings.Contains(out, "system:authenticated:oauth") {
		formattedDetailOut.WriteString("The self-provisioner role binding includes 'system:authenticated:oauth' group.\n\n")
		formattedDetailOut.WriteString("This allows all authenticated users to create new projects, which may lead to uncontrolled namespace proliferation.\n")
		formattedDetailOut.WriteString("In enterprise environments, it's often preferred to restrict project creation to administrators.\n\n")

		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Self-provisioner role binding includes system:authenticated:oauth, allowing uncontrolled namespace creation",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Remove the self-provisioner role from the system:authenticated:oauth group")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/building_applications/index#disabling-project-self-provisioning_configuring-project-creation", version))

		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// The self-provisioner role binding doesn't include system:authenticated:oauth
	formattedDetailOut.WriteString("The self-provisioner role binding is properly configured.\n\n")
	formattedDetailOut.WriteString("The 'system:authenticated:oauth' group is not included in the self-provisioners role binding,\n")
	formattedDetailOut.WriteString("which helps prevent uncontrolled namespace creation by regular users.\n\n")

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Self-provisioner role binding is properly configured to prevent uncontrolled namespace creation",
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailOut.String()
	return result, nil
}
