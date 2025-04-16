/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for UPI installations using MachineSets. It:

- Identifies if a UPI installation is using MachineSets for node management
- Checks if MachineSets are properly configured with replica counts
- Examines the integration between UPI installation and Machine API
- Provides insights into node management approaches
- Helps understand the cluster's scaling capabilities

This check helps identify the node management approach in UPI installations, which affects operational procedures.
*/

package cluster

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// UPIMachineSetCheck checks for user-provisioned infrastructure (UPI) using additional MachineSets
type UPIMachineSetCheck struct {
	healthcheck.BaseCheck
}

// NewUPIMachineSetCheck creates a new UPI MachineSets check
func NewUPIMachineSetCheck() *UPIMachineSetCheck {
	return &UPIMachineSetCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"upi-machinesets",
			"UPI with MachineSets",
			"Checks if user-provisioned infrastructure (UPI) is using additional MachineSets",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *UPIMachineSetCheck) Run() (healthcheck.Result, error) {
	// First determine if this is a UPI installation
	// Check for openshift-install configmap in openshift-config namespace
	// IPI typically has this configmap, UPI typically doesn't
	cmOutput, cmErr := utils.RunCommand("oc", "get", "cm", "-n", "openshift-config", "openshift-install")
	isIPI := cmErr == nil && strings.Contains(cmOutput, "openshift-install")

	// Format the detailed output with proper AsciiDoc formatting
	var formattedDetailedOut strings.Builder
	formattedDetailedOut.WriteString("=== Installation Type Analysis ===\n\n")

	// Add installation type information
	if isIPI {
		formattedDetailedOut.WriteString("Installation Type: IPI (Installer-Provisioned Infrastructure)\n\n")
		formattedDetailedOut.WriteString("This check is not applicable to IPI installations.\n\n")
	} else {
		formattedDetailedOut.WriteString("Installation Type: UPI (User-Provisioned Infrastructure)\n\n")
	}

	// If this is an IPI installation, this check is not applicable
	if isIPI {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"This is an IPI installation, UPI MachineSets check is not applicable",
			types.ResultKeyNotApplicable,
		), nil
	}

	// Check platform type
	platformType, err := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platform}")
	if err != nil || platformType == "" {
		// Try alternative path in newer versions
		platformType, err = utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.type}")
		if err != nil {
			return healthcheck.NewResult(
				c.ID(),
				types.StatusCritical,
				"Failed to get infrastructure platform type",
				types.ResultKeyRequired,
			), fmt.Errorf("error getting infrastructure platform type: %v", err)
		}
	}

	// Add platform type information
	formattedDetailedOut.WriteString(fmt.Sprintf("Platform Type: %s\n\n", strings.TrimSpace(platformType)))

	// If the platform is None (AnyPlatform), check for control plane topology
	isHCP := false
	if strings.TrimSpace(platformType) == "None" {
		controlPlaneTopology, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.controlPlaneTopology}")
		isHCP = strings.TrimSpace(controlPlaneTopology) == "External"

		if isHCP {
			formattedDetailedOut.WriteString("Control Plane Topology: External (Hosted Control Plane)\n\n")
		} else {
			formattedDetailedOut.WriteString("Control Plane Topology: Integrated\n\n")
		}
	}

	// Check for machinesets in openshift-machine-api namespace
	msOutput, msErr := utils.RunCommand("oc", "get", "machinesets", "-n", "openshift-machine-api")
	hasMachineSets := msErr == nil && strings.TrimSpace(msOutput) != ""

	// Check machine sets replica count
	msReplicaOutput, _ := utils.RunCommand("oc", "get", "machinesets.machine.openshift.io", "-n", "openshift-machine-api", "-o", "jsonpath={.items[*].spec.replicas}")
	hasMachineSetReplicas := msReplicaOutput != "" && strings.TrimSpace(msReplicaOutput) != ""

	// Add MachineSets information
	if hasMachineSets && strings.TrimSpace(msOutput) != "" {
		formattedDetailedOut.WriteString("MachineSets Found:\n[source, bash]\n----\n")
		formattedDetailedOut.WriteString(msOutput)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("MachineSets: None found\n\n")
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "machinesets", "-n", "openshift-machine-api", "-o", "yaml")
	if err == nil && strings.TrimSpace(detailedOut) != "" {
		formattedDetailedOut.WriteString("MachineSets Detailed Configuration:\n[source, yaml]\n----\n")
		formattedDetailedOut.WriteString(detailedOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	}

	// If this is an HCP cluster, the check is not applicable
	if isHCP {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"This is a Hosted Control Plane (HCP) installation, UPI MachineSets check is not applicable",
			types.ResultKeyNotApplicable,
		), nil
	}

	// If no machine sets exist in a UPI installation, inform but don't mark as an issue
	if !hasMachineSets {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"UPI installation without MachineSets - using traditional node management",
			types.ResultKeyNoChange,
		), nil
	}

	// If machine sets exist but no replicas are defined, this might be an issue
	if hasMachineSets && !hasMachineSetReplicas {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"UPI installation has MachineSets defined but without replica counts",
			types.ResultKeyAdvisory,
		)
		result.AddRecommendation("Check if MachineSets are properly configured with replica counts")
		result.AddRecommendation("If MachineSets are not being used, consider removing them to avoid confusion")
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// Get more detailed machineset information for the report
	machineSetCount, _ := utils.RunCommand("oc", "get", "machinesets", "-n", "openshift-machine-api", "-o", "name", "|", "wc", "-l")
	machineSetCount = strings.TrimSpace(machineSetCount)

	machineSetSummary, _ := utils.RunCommand("oc", "get", "machinesets", "-n", "openshift-machine-api", "-o", "custom-columns=NAME:.metadata.name,REPLICAS:.spec.replicas")

	// Add MachineSets summary information
	if strings.TrimSpace(machineSetSummary) != "" {
		formattedDetailedOut.WriteString("MachineSets Summary:\n[source, bash]\n----\n")
		formattedDetailedOut.WriteString(machineSetSummary)
		formattedDetailedOut.WriteString("\n----\n\n")
	}

	// Machine sets are properly configured in a UPI installation
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("UPI installation with %s configured MachineSets", machineSetCount),
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailedOut.String()
	return result, nil
}
