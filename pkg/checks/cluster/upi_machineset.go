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

	// If the platform is None (AnyPlatform), check for control plane topology
	isHCP := false
	if strings.TrimSpace(platformType) == "None" {
		controlPlaneTopology, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.controlPlaneTopology}")
		isHCP = strings.TrimSpace(controlPlaneTopology) == "External"
	}

	// Check for machinesets in openshift-machine-api namespace
	msOutput, msErr := utils.RunCommand("oc", "get", "machinesets", "-n", "openshift-machine-api")
	hasMachineSets := msErr == nil && strings.TrimSpace(msOutput) != ""

	// Check machine sets replica count
	msReplicaOutput, _ := utils.RunCommand("oc", "get", "machinesets.machine.openshift.io", "-n", "openshift-machine-api", "-o", "jsonpath={.items[*].spec.replicas}")
	hasMachineSetReplicas := msReplicaOutput != "" && strings.TrimSpace(msReplicaOutput) != ""

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "machinesets", "-n", "openshift-machine-api", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed machineset configuration"
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
		result.Detail = detailedOut
		return result, nil
	}

	// Get more detailed machineset information for the report
	machineSetCount, _ := utils.RunCommand("oc", "get", "machinesets", "-n", "openshift-machine-api", "-o", "name", "|", "wc", "-l")
	machineSetCount = strings.TrimSpace(machineSetCount)

	machineSetSummary, _ := utils.RunCommand("oc", "get", "machinesets", "-n", "openshift-machine-api", "-o", "custom-columns=NAME:.metadata.name,REPLICAS:.spec.replicas")

	// Machine sets are properly configured in a UPI installation
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("UPI installation with %s configured MachineSets", machineSetCount),
		types.ResultKeyNoChange,
	)
	result.Detail = fmt.Sprintf("MachineSets summary:\n\n%s\n\nDetailed configuration:\n%s", machineSetSummary, detailedOut)
	return result, nil
}
