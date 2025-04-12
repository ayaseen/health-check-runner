package cluster

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
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
			healthcheck.CategoryCluster,
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
			healthcheck.StatusCritical,
			"Failed to get Kubernetes client",
			healthcheck.ResultKeyRequired,
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
			healthcheck.StatusCritical,
			"Failed to retrieve control plane nodes",
			healthcheck.ResultKeyRequired,
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
			healthcheck.StatusOK,
			"All control plane nodes are properly configured to prevent regular workloads",
			healthcheck.ResultKeyNoChange,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// Create result with schedulable control node information
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusWarning,
		fmt.Sprintf("%d control plane nodes allow scheduling of regular workloads: %s",
			len(schedulableControlNodes), strings.Join(schedulableControlNodes, ", ")),
		healthcheck.ResultKeyRecommended,
	)

	result.AddRecommendation("Control plane nodes should be dedicated to control plane components for better reliability")
	result.AddRecommendation("Add NoSchedule taints to control plane nodes using 'oc adm taint nodes <node-name> node-role.kubernetes.io/master=:NoSchedule'")
	result.AddRecommendation("Alternatively, mark control plane nodes as unschedulable using 'oc adm cordon <node-name>'")

	result.Detail = detailedOut
	return result, nil
}

// WorkloadOffInfraNodesCheck checks if workloads are scheduled on infrastructure nodes
type WorkloadOffInfraNodesCheck struct {
	healthcheck.BaseCheck
}

// NewWorkloadOffInfraNodesCheck creates a new workload off infrastructure nodes check
func NewWorkloadOffInfraNodesCheck() *WorkloadOffInfraNodesCheck {
	return &WorkloadOffInfraNodesCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"workload-off-infra-nodes",
			"Workloads on Infrastructure Nodes",
			"Checks if user workloads are scheduled on infrastructure nodes",
			healthcheck.CategoryCluster,
		),
	}
}

// Run executes the health check
func (c *WorkloadOffInfraNodesCheck) Run() (healthcheck.Result, error) {
	// Get Kubernetes clientset
	clientset, err := utils.GetClientSet()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get Kubernetes client",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting Kubernetes client: %v", err)
	}

	// Get the list of infrastructure nodes
	ctx := context.Background()
	infraNodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/infra=",
	})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to retrieve infrastructure nodes",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error retrieving infrastructure nodes: %v", err)
	}

	// If no infrastructure nodes exist, this check is not applicable
	if len(infraNodes.Items) == 0 {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusNotApplicable,
			"No dedicated infrastructure nodes found in the cluster",
			healthcheck.ResultKeyNotApplicable,
		), nil
	}

	// Get all pods in user namespaces and check if they are scheduled on infrastructure nodes
	var podsOnInfraNodes []string
	var namespaces []string

	// Get all namespaces
	allNamespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to retrieve namespaces",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error retrieving namespaces: %v", err)
	}

	// Filter out system namespaces
	for _, ns := range allNamespaces.Items {
		if !strings.HasPrefix(ns.Name, "openshift-") &&
			ns.Name != "default" && ns.Name != "kube-system" &&
			ns.Name != "kube-public" && ns.Name != "kube-node-lease" {
			namespaces = append(namespaces, ns.Name)
		}
	}

	// Create a map of infrastructure node names for faster lookup
	infraNodeMap := make(map[string]bool)
	for _, node := range infraNodes.Items {
		infraNodeMap[node.Name] = true
	}

	// Check pods in user namespaces
	for _, ns := range namespaces {
		pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Non-critical error, we can continue with other namespaces
			continue
		}

		for _, pod := range pods.Items {
			// Skip pods that are being terminated
			if pod.DeletionTimestamp != nil {
				continue
			}

			// Check if the pod is running on an infrastructure node
			if infraNodeMap[pod.Spec.NodeName] {
				podsOnInfraNodes = append(podsOnInfraNodes,
					fmt.Sprintf("- Pod '%s' in namespace '%s' is running on infrastructure node '%s'",
						pod.Name, pod.Namespace, pod.Spec.NodeName))
			}
		}
	}

	// Get detailed node information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=", "-o", "wide")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed infrastructure node information"
	}

	// If no user workloads are running on infrastructure nodes, the check passes
	if len(podsOnInfraNodes) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			"No user workloads are running on infrastructure nodes",
			healthcheck.ResultKeyNoChange,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// Create result with workloads on infrastructure nodes information
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusWarning,
		fmt.Sprintf("%d user workloads are running on infrastructure nodes", len(podsOnInfraNodes)),
		healthcheck.ResultKeyRecommended,
	)

	result.AddRecommendation("Infrastructure nodes should be dedicated to infrastructure components")
	result.AddRecommendation("Add taints to infrastructure nodes to prevent user workloads from running on them")
	result.AddRecommendation("Consider moving these workloads to worker nodes")

	// Add detailed information
	detail := fmt.Sprintf("User workloads running on infrastructure nodes:\n%s\n\nInfrastructure nodes:\n%s",
		strings.Join(podsOnInfraNodes, "\n"),
		detailedOut)

	result.Detail = detail
	return result, nil
}

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
			healthcheck.CategoryCluster,
		),
	}
}

// Run executes the health check
func (c *InstallationTypeCheck) Run() (healthcheck.Result, error) {
	// Get the installation type
	out, err := utils.RunCommand("oc", "get", "infrastructure", "cluster",
		"-o", "jsonpath={.status.infrastructureName}")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check installation type",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking installation type: %v", err)
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed infrastructure configuration"
	}

	infrastructureName := strings.TrimSpace(out)
	if infrastructureName == "" {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"No infrastructure name detected",
			healthcheck.ResultKeyRequired,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// Try to determine the installation type from the infrastructure name
	var installationType string
	if strings.Contains(infrastructureName, "-upi-") {
		installationType = "User-Provisioned Infrastructure (UPI)"
	} else if strings.Contains(infrastructureName, "-ipi-") {
		installationType = "Installer-Provisioned Infrastructure (IPI)"
	} else {
		installationType = "Unknown"
	}

	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		fmt.Sprintf("Installation type: %s", installationType),
		healthcheck.ResultKeyNoChange,
	)
	result.Detail = fmt.Sprintf("Infrastructure Name: %s\n\n%s", infrastructureName, detailedOut)
	return result, nil
}
