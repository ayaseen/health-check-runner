/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for storage performance characteristics. It:

- Examines storage class configurations for performance annotations
- Identifies storage classes with performance tiers
- Provides recommendations for storage class organization
- Helps ensure appropriate storage performance for different workloads
- Identifies potential storage performance bottlenecks

This check helps administrators configure and maintain storage resources that meet application performance requirements.
*/

package storage

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// StoragePerformanceCheck assesses storage performance characteristics
type StoragePerformanceCheck struct {
	healthcheck.BaseCheck
}

// NewStoragePerformanceCheck creates a new storage performance check
func NewStoragePerformanceCheck() *StoragePerformanceCheck {
	return &StoragePerformanceCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"storage-performance",
			"Storage Performance",
			"Assesses storage performance characteristics",
			types.CategoryStorage,
		),
	}
}

// Run executes the health check
func (c *StoragePerformanceCheck) Run() (healthcheck.Result, error) {
	// Get storage classes
	detailedOut, err := utils.RunCommand("oc", "get", "storageclasses", "-o", "yaml")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to retrieve storage classes",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving storage classes: %v", err)
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Storage Performance Analysis ===\n\n")

	// Add storage classes information with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut.WriteString("Storage Classes Configuration:\n[source, yaml]\n----\n")
		formattedDetailOut.WriteString(detailedOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Storage Classes Configuration: No information available\n\n")
	}

	// Get storage classes in a more readable format for analysis
	tableOut, _ := utils.RunCommand("oc", "get", "storageclasses")
	if strings.TrimSpace(tableOut) != "" {
		formattedDetailOut.WriteString("Storage Classes Summary:\n[source, bash]\n----\n")
		formattedDetailOut.WriteString(tableOut)
		formattedDetailOut.WriteString("\n----\n\n")
	}

	// Check if we have any storage classes with performance annotations
	hasPerformanceClasses := strings.Contains(detailedOut, "performance") ||
		strings.Contains(detailedOut, "iops") ||
		strings.Contains(detailedOut, "throughput")

	// Add performance analysis section
	formattedDetailOut.WriteString("=== Performance Analysis ===\n\n")

	if hasPerformanceClasses {
		formattedDetailOut.WriteString("Performance-related annotations found in storage classes.\n\n")

		// Extract performance-related strings for more detailed information
		if strings.Contains(detailedOut, "performance") {
			formattedDetailOut.WriteString("- Found 'performance' annotations\n")
		}
		if strings.Contains(detailedOut, "iops") {
			formattedDetailOut.WriteString("- Found 'iops' annotations\n")
		}
		if strings.Contains(detailedOut, "throughput") {
			formattedDetailOut.WriteString("- Found 'throughput' annotations\n")
		}
		formattedDetailOut.WriteString("\n")
	} else {
		formattedDetailOut.WriteString("No explicit performance-related annotations found in storage classes.\n\n")
		formattedDetailOut.WriteString("Performance-related annotations might include:\n")
		formattedDetailOut.WriteString("- 'performance' - General performance tier indicators\n")
		formattedDetailOut.WriteString("- 'iops' - Input/Output Operations Per Second guarantees\n")
		formattedDetailOut.WriteString("- 'throughput' - Data transfer rate guarantees\n\n")
	}

	// Add best practices section
	formattedDetailOut.WriteString("=== Storage Performance Best Practices ===\n\n")
	formattedDetailOut.WriteString("1. Define different storage classes for different performance needs\n")
	formattedDetailOut.WriteString("2. Consider creating tiers like 'high', 'medium', and 'standard'\n")
	formattedDetailOut.WriteString("3. Label storage classes with performance characteristics\n")
	formattedDetailOut.WriteString("4. Document IOPS and throughput expectations for each class\n")
	formattedDetailOut.WriteString("5. Monitor storage performance regularly\n\n")

	if !hasPerformanceClasses {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No storage classes with explicit performance characteristics found",
			types.ResultKeyAdvisory,
		)

		result.AddRecommendation("Consider defining storage classes with different performance tiers")
		result.AddRecommendation("Label storage classes with performance characteristics for better workload placement")

		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Storage classes with performance characteristics are available",
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailOut.String()
	return result, nil
}
