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
	// This is a placeholder for a more comprehensive storage performance check
	// In a production environment, this would likely include:
	// - Analysis of storage class QoS parameters
	// - Assessment of actual performance via metrics
	// - Evaluation of appropriate storage classes for different workloads

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

	// Check if we have any storage classes with performance annotations
	hasPerformanceClasses := strings.Contains(detailedOut, "performance") ||
		strings.Contains(detailedOut, "iops") ||
		strings.Contains(detailedOut, "throughput")

	if !hasPerformanceClasses {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No storage classes with explicit performance characteristics found",
			types.ResultKeyAdvisory,
		)

		result.AddRecommendation("Consider defining storage classes with different performance tiers")
		result.AddRecommendation("Label storage classes with performance characteristics for better workload placement")

		result.Detail = fmt.Sprintf("Storage Class Details:\n%s", detailedOut)

		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"Storage classes with performance characteristics are available",
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
