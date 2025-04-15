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

	// Check machine sets replica count - used for UPI with machinesets
	msReplicaOutput, _ := utils.RunCommand("oc", "get", "machinesets.machine.openshift.io", "-n", "openshift-machine-api", "-o", "jsonpath={.items[*].spec.replicas}")
	hasMachineSetReplicas := msReplicaOutput != "" && strings.TrimSpace(msReplicaOutput) != ""

	// Get the infrastructure name for detail information
	infraName, err := utils.RunCommand("oc", "get", "infrastructure", "cluster",
		"-o", "jsonpath={.status.infrastructureName}")
	if err != nil {
		// Non-critical error for the infrastructure name
		infraName = "Unable to determine infrastructure name"
	}

	// Get infrastructure platform type
	platformType, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster",
		"-o", "jsonpath={.status.platform}")
	if platformType == "" {
		// Try alternative path for newer versions
		platformType, _ = utils.RunCommand("oc", "get", "infrastructure", "cluster",
			"-o", "jsonpath={.status.platformStatus.type}")
	}

	// Get control plane and infrastructure topology
	controlPlaneTopology, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster",
		"-o", "jsonpath={.status.controlPlaneTopology}")
	infrastructureTopology, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster",
		"-o", "jsonpath={.status.infrastructureTopology}")

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed infrastructure configuration"
	}

	// Determine installation type based on our checks
	var installationType string
	var message string
	var description string

	// Check if it's AnyPlatform/None
	if strings.EqualFold(platformType, "None") {
		if controlPlaneTopology == "External" {
			installationType = "Hosted Control Plane (HCP)"
			message = "Installation type: Hosted Control Plane (HCP)"
			description = "This is a Hosted Control Plane (HCP) installation where the control plane components run externally to the cluster."
		} else {
			installationType = "User-Provisioned Infrastructure (UPI) with no cloud provider"
			message = "Installation type: UPI (No Cloud Provider)"
			description = "This is a UPI installation with no specific cloud provider integration."
		}
	} else if isIPI {
		installationType = "Installer-Provisioned Infrastructure (IPI)"
		message = "Installation type: IPI"
		description = "This is an IPI installation where the OpenShift installer provisioned the infrastructure automatically."
	} else if hasMachineSets && hasMachineSetReplicas {
		// If we have machinesets with replicas but no openshift-install configmap, it might be UPI with machine-api
		installationType = "User-Provisioned Infrastructure (UPI) with Machine API integration"
		message = "Installation type: UPI with Machine API"
		description = "This appears to be a UPI installation that leverages Machine API for node lifecycle management."
	} else {
		installationType = "User-Provisioned Infrastructure (UPI)"
		message = "Installation type: UPI"
		description = "This is a UPI installation where the infrastructure was provisioned manually or by external automation."
	}

	// Include topology information in the message if available
	var topologyInfo string
	if controlPlaneTopology != "" && infrastructureTopology != "" {
		topologyInfo = fmt.Sprintf("\nControl Plane Topology: %s\nInfrastructure Topology: %s",
			controlPlaneTopology, infrastructureTopology)
	}

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		message,
		types.ResultKeyNoChange,
	)

	// Add detailed information to the result
	result.Detail = fmt.Sprintf("Infrastructure Name: %s\nPlatform Type: %s\n\nInstallation Type: %s\n\n%s%s\n\n%s",
		strings.TrimSpace(infraName),
		strings.TrimSpace(platformType),
		installationType,
		description,
		topologyInfo,
		detailedOut)

	return result, nil
}
