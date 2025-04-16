/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for infrastructure provider configuration. It:

- Identifies the underlying infrastructure provider (AWS, Azure, GCP, etc.)
- Gathers provider-specific information for context
- Examines control plane and infrastructure topology
- Provides insights into the cluster's infrastructure setup
- Helps understand the deployment environment for proper troubleshooting

This check provides important context about the underlying platform hosting the OpenShift cluster.
*/

package cluster

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// InfrastructureProviderCheck checks the infrastructure provider configuration
type InfrastructureProviderCheck struct {
	healthcheck.BaseCheck
}

// NewInfrastructureProviderCheck creates a new infrastructure provider check
func NewInfrastructureProviderCheck() *InfrastructureProviderCheck {
	return &InfrastructureProviderCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"infrastructure-provider",
			"Infrastructure Provider",
			"Checks the infrastructure provider configuration",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *InfrastructureProviderCheck) Run() (healthcheck.Result, error) {
	// Get the infrastructure provider type from primary path
	platformType, err := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platform}")
	if err != nil || platformType == "" {
		// Try alternative path in newer versions
		platformType, err = utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.type}")
		if err != nil {
			return healthcheck.NewResult(
				c.ID(),
				types.StatusCritical,
				"Failed to get infrastructure provider",
				types.ResultKeyRequired,
			), fmt.Errorf("error getting infrastructure provider: %v", err)
		}
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed infrastructure configuration"
	}

	// Get additional provider-specific information based on the platform type
	var providerInfo string
	var additionalInfo string

	platformType = strings.TrimSpace(platformType)

	// Get infrastructure name
	infraName, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.infrastructureName}")
	infraName = strings.TrimSpace(infraName)

	// Get control plane and infrastructure topology
	controlPlaneTopology, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.controlPlaneTopology}")
	infrastructureTopology, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.infrastructureTopology}")

	switch platformType {
	case "AWS":
		// Get AWS region
		region, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.aws.region}")
		providerInfo = fmt.Sprintf("AWS Region: %s", strings.TrimSpace(region))

		// Check for custom service endpoints
		hasCustomEndpoints, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.aws.serviceEndpoints}")
		if strings.TrimSpace(hasCustomEndpoints) != "" && strings.TrimSpace(hasCustomEndpoints) != "[]" {
			additionalInfo = "Custom AWS service endpoints are configured."
		}
	case "Azure":
		// Get Azure resource group and cloud name
		resourceGroup, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.azure.resourceGroupName}")
		cloudName, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.azure.cloudName}")
		providerInfo = fmt.Sprintf("Azure Resource Group: %s", strings.TrimSpace(resourceGroup))
		if strings.TrimSpace(cloudName) != "" {
			providerInfo += fmt.Sprintf("\nAzure Cloud: %s", strings.TrimSpace(cloudName))
		}
	case "GCP":
		// Get GCP project ID and region
		projectID, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.gcp.projectID}")
		region, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.gcp.region}")
		providerInfo = fmt.Sprintf("GCP Project ID: %s\nGCP Region: %s",
			strings.TrimSpace(projectID), strings.TrimSpace(region))
	case "OpenStack":
		// Get OpenStack cloud name
		cloudName, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.openstack.cloudName}")
		providerInfo = fmt.Sprintf("OpenStack Cloud: %s", strings.TrimSpace(cloudName))
	case "BareMetal":
		// Check if it has a specific load balancer type
		loadBalancerType, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.baremetal.loadBalancer.type}")
		if strings.TrimSpace(loadBalancerType) != "" {
			providerInfo = fmt.Sprintf("Load Balancer Type: %s", strings.TrimSpace(loadBalancerType))
		} else {
			providerInfo = "Bare Metal infrastructure"
		}
	case "VSphere":
		// For vSphere, we might have multiple vCenters
		vCenters, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.spec.platformSpec.vsphere.vcenters[*].server}")
		if strings.TrimSpace(vCenters) != "" {
			providerInfo = fmt.Sprintf("vSphere vCenters: %s", strings.TrimSpace(vCenters))
		} else {
			providerInfo = "vSphere infrastructure"
		}
	case "oVirt":
		providerInfo = "oVirt infrastructure"
	case "IBMCloud":
		// Get IBM Cloud provider type (Classic, VPC, UPI)
		providerType, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.ibmcloud.providerType}")
		location, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.ibmcloud.location}")
		providerInfo = fmt.Sprintf("IBM Cloud Provider Type: %s\nLocation: %s",
			strings.TrimSpace(providerType), strings.TrimSpace(location))
	case "PowerVS":
		// Get Power VS region and zone
		region, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.powervs.region}")
		zone, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.powervs.zone}")
		providerInfo = fmt.Sprintf("PowerVS Region: %s\nZone: %s",
			strings.TrimSpace(region), strings.TrimSpace(zone))
	case "Nutanix":
		providerInfo = "Nutanix infrastructure"
	case "None":
		// For None/AnyPlatform, check if it's External topology (HCP)
		if controlPlaneTopology == "External" {
			providerInfo = "Hosted Control Plane (HCP) infrastructure"
		} else {
			providerInfo = "No specific cloud provider integration"
		}
	case "External":
		// For External platform, check if there's a platform name
		platformName, _ := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "jsonpath={.status.platformStatus.external.platformName}")
		if strings.TrimSpace(platformName) != "" && strings.TrimSpace(platformName) != "Unknown" {
			providerInfo = fmt.Sprintf("External Platform: %s", strings.TrimSpace(platformName))
		} else {
			providerInfo = "Generic external infrastructure provider"
		}
	default:
		providerInfo = fmt.Sprintf("Infrastructure provider: %s", platformType)
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Infrastructure Provider Information ===\n\n")

	// Add infrastructure basics
	formattedDetailOut.WriteString(fmt.Sprintf("Infrastructure Name: %s\n", infraName))
	formattedDetailOut.WriteString(fmt.Sprintf("Platform Type: %s\n\n", platformType))

	// Add provider-specific information
	formattedDetailOut.WriteString("Provider Details:\n")
	formattedDetailOut.WriteString(providerInfo)
	formattedDetailOut.WriteString("\n\n")

	// Add topology information if available
	if controlPlaneTopology != "" && infrastructureTopology != "" {
		formattedDetailOut.WriteString("Topology Information:\n")
		formattedDetailOut.WriteString(fmt.Sprintf("- Control Plane Topology: %s\n", controlPlaneTopology))
		formattedDetailOut.WriteString(fmt.Sprintf("- Infrastructure Topology: %s\n\n", infrastructureTopology))
	}

	// Add additional information if available
	if additionalInfo != "" {
		formattedDetailOut.WriteString("Additional Information:\n")
		formattedDetailOut.WriteString(additionalInfo)
		formattedDetailOut.WriteString("\n\n")
	}

	// Add raw configuration with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut.WriteString("Infrastructure Configuration:\n[source, yaml]\n----\n")
		formattedDetailOut.WriteString(detailedOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Infrastructure Configuration: No information available\n\n")
	}

	// If the platform type is empty, we'll mark it as critical
	if platformType == "" {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"No infrastructure provider type detected",
			types.ResultKeyRequired,
		)
		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Create result with the detected provider information
	message := fmt.Sprintf("Infrastructure provider type: %s", platformType)
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		message,
		types.ResultKeyNoChange,
	)

	result.Detail = formattedDetailOut.String()
	return result, nil
}
