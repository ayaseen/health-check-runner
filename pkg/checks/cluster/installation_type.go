package cluster

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// InstallationTypeCheck checks the installation type of OpenShift
type InstallationTypeCheck struct {
	healthcheck.BaseCheck
}

// NewInstallationTypeCheck creates a new installation type check
func NewInstallationTypeCheck() *InstallationTypeCheck {
	return &InstallationTypeCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"installation-type",
			"Installation Type",
			"Checks the installation type of OpenShift",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *InstallationTypeCheck) Run() (healthcheck.Result, error) {
	// First method: Check for openshift-install configmap in openshift-config namespace
	// This is a good indicator for IPI installations
	cmOutput, cmErr := utils.RunCommand("oc", "get", "cm", "-n", "openshift-config", "openshift-install")
	isIPI := cmErr == nil && strings.Contains(cmOutput, "openshift-install")

	// Second method: Check for machinesets in openshift-machine-api namespace
	// IPI typically has machinesets, UPI might not
	msOutput, msErr := utils.RunCommand("oc", "get", "machinesets", "-n", "openshift-machine-api")
	hasMachineSets := msErr == nil && strings.TrimSpace(msOutput) != ""

	// Get the infrastructure name for detail information
	infraName, err := utils.RunCommand("oc", "get", "infrastructure", "cluster",
		"-o", "jsonpath={.status.infrastructureName}")
	if err != nil {
		// Non-critical error for the infrastructure name
		infraName = "Unable to determine infrastructure name"
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed infrastructure configuration"
	}

	// Determine installation type based on our checks
	var installationType string
	var message string

	if isIPI {
		installationType = "Installer-Provisioned Infrastructure (IPI)"
		message = "Installation type: IPI"
	} else if hasMachineSets {
		// If we have machinesets but no openshift-install configmap, it's likely still IPI
		// but the configmap might have been removed
		installationType = "Likely Installer-Provisioned Infrastructure (IPI)"
		message = "Installation type: Likely IPI (has machinesets but missing confirming configmap)"
	} else {
		installationType = "User-Provisioned Infrastructure (UPI)"
		message = "Installation type: UPI"
	}

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		message,
		types.ResultKeyNoChange,
	)

	// Add detailed information to the result
	result.Detail = fmt.Sprintf("Infrastructure Name: %s\n\nInstallation Type: %s\n\nMachinesets present: %t\n\n%s",
		strings.TrimSpace(infraName),
		installationType,
		hasMachineSets,
		detailedOut)

	return result, nil
}
