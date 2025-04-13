package cluster

import (
	"context"
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// NodeStatusCheck checks if all nodes are ready
type NodeStatusCheck struct {
	healthcheck.BaseCheck
}

// NewNodeStatusCheck creates a new node status check
func NewNodeStatusCheck() *NodeStatusCheck {
	return &NodeStatusCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"node-status",
			"Node Status",
			"Checks if all nodes are ready",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *NodeStatusCheck) Run() (healthcheck.Result, error) {
	// Get Kubernetes clientset
	clientset, err := utils.GetClientSet()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get Kubernetes client",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting Kubernetes client: %v", err)
	}

	// Get the list of nodes
	ctx := context.Background()
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to retrieve nodes",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving nodes: %v", err)
	}

	// Check node status
	var notReadyNodes []string

	for _, node := range nodes.Items {
		nodeReady := false

		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				nodeReady = true
				break
			}
		}

		if !nodeReady {
			notReadyNodes = append(notReadyNodes, node.Name)
		}
	}

	// Get the output of 'oc get nodes' for detailed information
	detailedOut, err := utils.RunCommand("oc", "get", "nodes")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed node status"
	}

	if len(notReadyNodes) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			fmt.Sprintf("All %d nodes are ready", len(nodes.Items)),
			types.ResultKeyNoChange,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// Create result with not ready nodes information
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusCritical,
		fmt.Sprintf("%d nodes are not ready: %s", len(notReadyNodes), strings.Join(notReadyNodes, ", ")),
		types.ResultKeyRequired,
	)

	result.AddRecommendation("Investigate why the nodes are not ready")
	result.AddRecommendation("Check node logs using 'oc adm node-logs <node-name>'")
	result.AddRecommendation("Check node diagnostics using 'oc debug node/<node-name>'")

	result.Detail = detailedOut
	return result, nil
}

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
