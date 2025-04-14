package cluster

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// ControlNodeSchedulableCheck checks if control plane nodes are marked as unschedulable for workloads
type ControlNodeSchedulableCheck struct {
	healthcheck.BaseCheck
}

// NewControlNodeSchedulableCheck creates a new control node schedulable check
func NewControlNodeSchedulableCheck() *ControlNodeSchedulableCheck {
	return &ControlNodeSchedulableCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"control-node-schedulable",
			"Control Plane Node Schedulability",
			"Checks if control plane nodes are marked as unschedulable for regular workloads",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *ControlNodeSchedulableCheck) Run() (healthcheck.Result, error) {
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
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/master=",
	})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to retrieve control plane nodes",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving control plane nodes: %v", err)
	}

	// Check if any control plane nodes are schedulable
	var schedulableControlNodes []string

	for _, node := range nodes.Items {
		if !node.Spec.Unschedulable {
			// Check if there's a taint that prevents regular workloads
			hasTaint := false
			for _, taint := range node.Spec.Taints {
				if taint.Key == "node-role.kubernetes.io/master" &&
					(taint.Effect == "NoSchedule" || taint.Effect == "NoExecute") {
					hasTaint = true
					break
				}
			}

			if !hasTaint {
				schedulableControlNodes = append(schedulableControlNodes, node.Name)
			}
		}
	}

	// Get detailed node information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/master=", "-o", "wide")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed control plane node information"
	}

	if len(schedulableControlNodes) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"All control plane nodes are properly configured to prevent regular workloads",
			types.ResultKeyNoChange,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// Create result with schedulable control node information
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		fmt.Sprintf("%d control plane nodes allow scheduling of regular workloads: %s",
			len(schedulableControlNodes), strings.Join(schedulableControlNodes, ", ")),
		types.ResultKeyRecommended,
	)

	result.AddRecommendation("Control plane nodes should be dedicated to control plane components for better reliability")
	result.AddRecommendation("Add NoSchedule taints to control plane nodes using 'oc adm taint nodes <node-name> node-role.kubernetes.io/master=:NoSchedule'")
	result.AddRecommendation("Alternatively, mark control plane nodes as unschedulable using 'oc adm cordon <node-name>'")

	result.Detail = detailedOut
	return result, nil
}
