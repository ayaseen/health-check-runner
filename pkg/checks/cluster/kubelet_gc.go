/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for kubelet garbage collection configuration. It:

- Examines kubelet configuration for garbage collection settings
- Checks for container and image cleanup policies
- Verifies log rotation and maximum size settings
- Identifies nodes with storage issues
- Provides recommendations for optimal garbage collection configuration

This check helps prevent node storage issues by ensuring proper cleanup of unused containers and images.
*/

package cluster

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// KubeletGarbageCollectionCheck checks if kubelet garbage collection is properly configured
type KubeletGarbageCollectionCheck struct {
	healthcheck.BaseCheck
	cpuThreshold    int
	memoryThreshold int
}

// NewKubeletGarbageCollectionCheck creates a new kubelet garbage collection check
func NewKubeletGarbageCollectionCheck() *KubeletGarbageCollectionCheck {
	return &KubeletGarbageCollectionCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"kubelet-garbage-collection",
			"Kubelet Garbage Collection",
			"Checks if kubelet garbage collection is properly configured",
			types.CategoryClusterConfig,
		),
		cpuThreshold:    80, // 80% CPU usage threshold
		memoryThreshold: 80, // 80% memory usage threshold
	}
}

// Run executes the health check
func (c *KubeletGarbageCollectionCheck) Run() (healthcheck.Result, error) {
	// Check for kubelet config
	out, err := utils.RunCommand("oc", "get", "kubeletconfig", "-o", "name")

	kubeletConfigExists := err == nil && strings.TrimSpace(out) != ""

	// Get detailed information for the report
	var detailedOut string
	if kubeletConfigExists {
		detailedCmd, err := utils.RunCommand("oc", "get", strings.TrimSpace(out), "-o", "yaml")
		if err == nil {
			detailedOut = detailedCmd
		} else {
			detailedOut = "Failed to get detailed kubelet configuration"
		}
	} else {
		detailedOut = "No custom kubelet configuration found"
	}

	// Format the detailed output with proper AsciiDoc formatting
	var formattedDetailedOut strings.Builder
	formattedDetailedOut.WriteString("=== Kubelet Garbage Collection Analysis ===\n\n")

	// Add kubelet configuration information with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailedOut.WriteString("Kubelet Configuration:\n[source, yaml]\n----\n")
		formattedDetailedOut.WriteString(detailedOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Kubelet Configuration: No information available\n\n")
	}

	// Check for garbage collection settings
	gcThreshold, err := utils.RunCommand("oc", "get", "kubeletconfigs.machineconfiguration.openshift.io", "-o", "jsonpath={.items[*].spec.kubeletConfig.evictionHard}")

	gcConfigured := err == nil && strings.TrimSpace(gcThreshold) != ""

	// Get machine config pools with kubelet config - store for detailed reporting
	mcpInfo, err := utils.RunCommand("oc", "get", "mcp", "-o", "jsonpath={.items[*].metadata.name}")
	if err == nil && strings.TrimSpace(mcpInfo) != "" {
		formattedDetailedOut.WriteString("Machine Config Pools:\n[source, text]\n----\n")
		formattedDetailedOut.WriteString(mcpInfo)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Machine Config Pools: No information available\n\n")
	}

	// Get node storage information to check for potential issues
	nodeStorageOut, err := utils.RunCommand("oc", "adm", "top", "nodes", "|", "grep", "100%")

	if err == nil && strings.TrimSpace(nodeStorageOut) != "" {
		formattedDetailedOut.WriteString("Node Storage Issues Detected:\n[source, bash]\n----\n")
		formattedDetailedOut.WriteString(nodeStorageOut)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Node Storage Issues: None detected\n\n")
	}

	nodeStorageIssues := err == nil && strings.TrimSpace(nodeStorageOut) != ""

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Check for image garbage collection settings
	imageGCOut, err := utils.RunCommand("oc", "get", "kubeletconfigs.machineconfiguration.openshift.io", "-o", "jsonpath={.items[*].spec.kubeletConfig.imageGCHighThresholdPercent}")

	imageGCConfigured := err == nil && strings.TrimSpace(imageGCOut) != ""

	// Check for container log max size and max files
	containerLogOut, err := utils.RunCommand("oc", "get", "kubeletconfigs.machineconfiguration.openshift.io", "-o", "jsonpath={.items[*].spec.kubeletConfig.containerLogMaxSize}")

	containerLogConfigured := err == nil && strings.TrimSpace(containerLogOut) != ""

	// Add configuration status section to formatted output
	formattedDetailedOut.WriteString("=== Configuration Status ===\n\n")
	formattedDetailedOut.WriteString(fmt.Sprintf("Custom Kubelet Configuration Exists: %v\n", kubeletConfigExists))
	formattedDetailedOut.WriteString(fmt.Sprintf("Garbage Collection Thresholds Configured: %v\n", gcConfigured))
	formattedDetailedOut.WriteString(fmt.Sprintf("Image Garbage Collection Configured: %v\n", imageGCConfigured))
	formattedDetailedOut.WriteString(fmt.Sprintf("Container Log Size Limits Configured: %v\n", containerLogConfigured))
	formattedDetailedOut.WriteString(fmt.Sprintf("Node Storage Issues Detected: %v\n\n", nodeStorageIssues))

	// Evaluate kubelet garbage collection configuration
	if !kubeletConfigExists {
		// No custom kubelet config exists, check if we're seeing storage issues
		if nodeStorageIssues {
			result := healthcheck.NewResult(
				c.ID(),
				types.StatusWarning,
				"No custom kubelet garbage collection configuration found and node storage issues detected",
				types.ResultKeyRecommended,
			)
			result.AddRecommendation("Configure kubelet garbage collection parameters to prevent node storage issues")
			result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/nodes/index#nodes-nodes-garbage-collection", version))
			result.Detail = formattedDetailedOut.String()
			return result, nil
		}

		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No custom kubelet garbage collection configuration found (using defaults)",
			types.ResultKeyAdvisory,
		)
		result.AddRecommendation("Consider configuring kubelet garbage collection parameters for production environments")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/nodes/index#nodes-nodes-garbage-collection", version))
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// Custom kubelet config exists, but garbage collection may not be configured
	if !gcConfigured && !imageGCConfigured && !containerLogConfigured {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Custom kubelet configuration exists but garbage collection parameters are not set",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure evictionHard, imageGCHighThresholdPercent, and containerLogMaxSize parameters")
		result.AddRecommendation(fmt.Sprintf("Refer to the documentation at https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/nodes/index#nodes-nodes-garbage-collection", version))
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// Check if storage issues exist despite configuration
	if nodeStorageIssues {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"Kubelet garbage collection is configured but node storage issues are still present",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Review and adjust kubelet garbage collection thresholds")
		result.AddRecommendation("Check for specific workloads that might be causing excessive disk usage")
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// All checks passed
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Kubelet garbage collection is properly configured",
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailedOut.String()
	return result, nil
}
