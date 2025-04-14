package cluster

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
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
