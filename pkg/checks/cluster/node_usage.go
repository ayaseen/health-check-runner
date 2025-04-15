/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for node resource usage. It:

- Examines CPU and memory utilization across cluster nodes
- Identifies nodes with high resource consumption
- Analyzes resource trends and potential bottlenecks
- Provides recommendations for addressing resource constraints
- Helps ensure proper resource distribution and capacity planning

This check helps identify resource constraints before they impact application performance and availability.
*/

package cluster

import (
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// NodeUsageCheck checks node resource usage
type NodeUsageCheck struct {
	healthcheck.BaseCheck
	cpuThreshold    int
	memoryThreshold int
}

// NewNodeUsageCheck creates a new node usage check
func NewNodeUsageCheck() *NodeUsageCheck {
	return &NodeUsageCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"node-usage",
			"Node Usage",
			"Checks if nodes are within CPU and memory usage thresholds",
			types.CategoryClusterConfig,
		),
		cpuThreshold:    80, // 80% CPU usage threshold
		memoryThreshold: 80, // 80% memory usage threshold
	}
}

// Run executes the health check
func (c *NodeUsageCheck) Run() (healthcheck.Result, error) {
	// Get the output of 'oc adm top nodes' for resource usage information
	output, err := utils.RunCommand("oc", "adm", "top", "nodes")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get node usage information",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting node usage: %v", err)
	}

	// Parse the output to extract CPU and memory usage
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to parse node usage information",
			types.ResultKeyRequired,
		), fmt.Errorf("unexpected format of 'oc adm top nodes' output")
	}

	// Skip header line
	lines = lines[1:]

	// Check usage for each node
	var highCpuNodes []string
	var highMemoryNodes []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		nodeName := fields[0]

		// Parse CPU usage
		cpuUsage := strings.TrimSuffix(fields[2], "%")
		cpuUsageValue, err := parsePercentage(cpuUsage)
		if err == nil && cpuUsageValue > float64(c.cpuThreshold) {
			highCpuNodes = append(highCpuNodes, fmt.Sprintf("%s (%.2f%%)", nodeName, cpuUsageValue))
		}

		// Parse memory usage
		memoryUsage := strings.TrimSuffix(fields[4], "%")
		memoryUsageValue, err := parsePercentage(memoryUsage)
		if err == nil && memoryUsageValue > float64(c.memoryThreshold) {
			highMemoryNodes = append(highMemoryNodes, fmt.Sprintf("%s (%.2f%%)", nodeName, memoryUsageValue))
		}
	}

	if len(highCpuNodes) == 0 && len(highMemoryNodes) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"All nodes are within resource usage thresholds",
			types.ResultKeyNoChange,
		)
		result.Detail = output
		return result, nil
	}

	// Create result with high usage nodes information
	var message string
	resultKey := types.ResultKeyAdvisory

	if len(highCpuNodes) > 0 && len(highMemoryNodes) > 0 {
		message = fmt.Sprintf("%d nodes with high CPU usage and %d nodes with high memory usage",
			len(highCpuNodes), len(highMemoryNodes))
	} else if len(highCpuNodes) > 0 {
		message = fmt.Sprintf("%d nodes with high CPU usage", len(highCpuNodes))
	} else {
		message = fmt.Sprintf("%d nodes with high memory usage", len(highMemoryNodes))
	}

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		message,
		resultKey,
	)

	if len(highCpuNodes) > 0 {
		result.AddRecommendation(fmt.Sprintf("Investigate nodes with high CPU usage: %s", strings.Join(highCpuNodes, ", ")))
	}

	if len(highMemoryNodes) > 0 {
		result.AddRecommendation(fmt.Sprintf("Investigate nodes with high memory usage: %s", strings.Join(highMemoryNodes, ", ")))
	}

	result.AddRecommendation("Consider adding more nodes or optimizing workload placement")

	result.Detail = output
	return result, nil
}

// parsePercentage parses a percentage string (e.g., "85.2") to a float64
func parsePercentage(percentStr string) (float64, error) {
	var percentage float64
	_, err := fmt.Sscanf(percentStr, "%f", &percentage)
	return percentage, err
}
