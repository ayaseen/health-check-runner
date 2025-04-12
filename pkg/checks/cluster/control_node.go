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
		result = result.WithDetail(detailedOut)
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

	result = result.WithDetail(detailedOut)
	return result, nil
}

// InfrastructureNodesCheck checks if dedicated infrastructure nodes are configured
type InfrastructureNodesCheck struct {
	healthcheck.BaseCheck
}

// NewInfrastructureNodesCheck creates a new infrastructure nodes check
func NewInfrastructureNodesCheck() *InfrastructureNodesCheck {
	return &InfrastructureNodesCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"infrastructure-nodes",
			"Infrastructure Nodes",
			"Checks if dedicated infrastructure nodes are configured",
			healthcheck.CategoryCluster,
		),
	}
}

// Run executes the health check
func (c *InfrastructureNodesCheck) Run() (healthcheck.Result, error) {
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

	// Get the list of nodes with infrastructure role
	ctx := context.Background()
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
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

	// Get detailed node information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "nodes", "-o", "wide")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed node information"
	}

	if len(nodes.Items) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"No dedicated infrastructure nodes found",
			healthcheck.ResultKeyRecommended,
		)
		result = result.WithDetail(detailedOut)
		return result, nil
	}

	// Check if infrastructure nodes have the correct taints to prevent regular workloads
	var untaintedInfraNodes []string

	for _, node := range nodes.Items {
		hasTaint := false
		for _, taint := range node.Spec.Taints {
			if taint.Key == "node-role.kubernetes.io/infra" &&
				(taint.Effect == "NoSchedule" || taint.Effect == "NoExecute" || taint.Effect == "PreferNoSchedule") {
				hasTaint = true
				break
			}
		}

		if !hasTaint {
			untaintedInfraNodes = append(untaintedInfraNodes, node.Name)
		}
	}

	if len(untaintedInfraNodes) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			fmt.Sprintf("Found %d properly configured infrastructure nodes", len(nodes.Items)),
			healthcheck.ResultKeyNoChange,
		)
		result = result.WithDetail(detailedOut)
		return result, nil
	}

	// Create result with untainted infrastructure node information
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusWarning,
		fmt.Sprintf("%d infrastructure nodes are not tainted to prevent regular workloads: %s",
			len(untaintedInfraNodes), strings.Join(untaintedInfraNodes, ", ")),
		healthcheck.ResultKeyRecommended,
	)

	result.AddRecommendation("Infrastructure nodes should be dedicated to infrastructure components")
	result.AddRecommendation("Add NoSchedule taints to infrastructure nodes using 'oc adm taint nodes <node-name> node-role.kubernetes.io/infra=:NoSchedule'")

	result = result.WithDetail(detailedOut)
	return result, nil
}

// InfraMachineConfigPoolCheck checks if a dedicated infrastructure machine config pool exists
type InfraMachineConfigPoolCheck struct {
	healthcheck.BaseCheck
}

// NewInfraMachineConfigPoolCheck creates a new infrastructure machine config pool check
func NewInfraMachineConfigPoolCheck() *InfraMachineConfigPoolCheck {
	return &InfraMachineConfigPoolCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"infra-machine-config-pool",
			"Infrastructure Machine Config Pool",
			"Checks if a dedicated infrastructure machine config pool exists",
			healthcheck.CategoryCluster,
		),
	}
}

// Run executes the health check
func (c *InfraMachineConfigPoolCheck) Run() (healthcheck.Result, error) {
	// Check if the infrastructure machine config pool exists
	out, err := utils.RunCommand("oc", "get", "machineconfig", "50-infra")
	if err != nil {
		// This is not necessarily an error, as the infra MCP might not exist
		if strings.Contains(err.Error(), "not found") {
			return healthcheck.NewResult(
				c.ID(),
				healthcheck.StatusWarning,
				"No dedicated infrastructure machine config found",
				healthcheck.ResultKeyRecommended,
			), nil
		}

		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check for infrastructure machine config",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking for infrastructure machine config: %v", err)
	}

	// Check if the infrastructure machine config pool exists
	mcpOut, err := utils.RunCommand("oc", "get", "mcp", "infra")
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return healthcheck.NewResult(
				c.ID(),
				healthcheck.StatusWarning,
				"No dedicated infrastructure machine config pool found",
				healthcheck.ResultKeyRecommended,
			), nil
		}

		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check for infrastructure machine config pool",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking for infrastructure machine config pool: %v", err)
	}

	// If both resources exist, check if the machine config pool is properly configured
	detailedOut := fmt.Sprintf("Machine Config:\n%s\n\nMachine Config Pool:\n%s", out, mcpOut)

	// Check if the machine config pool is degraded
	degradedOut, err := utils.RunCommand("oc", "get", "mcp", "infra", "-o", "jsonpath={.status.conditions[?(@.type==\"Degraded\")].status}")
	if err != nil {
		// Non-critical error, we can continue
		degradedOut = "Failed to check if MCP is degraded"
	}

	if strings.TrimSpace(degradedOut) == "True" {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"Infrastructure machine config pool is degraded",
			healthcheck.ResultKeyRequired,
		)
		result = result.WithDetail(detailedOut)
		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		"Dedicated infrastructure machine config pool is properly configured",
		healthcheck.ResultKeyNoChange,
	)
	result = result.WithDetail(detailedOut)
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
		result = result.WithDetail(detailedOut)
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

	result = result.WithDetail(detail)
	return result, nil
}

// DefaultProjectTemplateCheck checks if a custom default project template is configured
type DefaultProjectTemplateCheck struct {
	healthcheck.BaseCheck
}

// NewDefaultProjectTemplateCheck creates a new default project template check
func NewDefaultProjectTemplateCheck() *DefaultProjectTemplateCheck {
	return &DefaultProjectTemplateCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"default-project-template",
			"Default Project Template",
			"Checks if a custom default project template is configured",
			healthcheck.CategoryCluster,
		),
	}
}

// Run executes the health check
func (c *DefaultProjectTemplateCheck) Run() (healthcheck.Result, error) {
	// Check if a custom default project template is configured
	out, err := utils.RunCommand("oc", "get", "projectrequests.config.openshift.io", "cluster",
		"-o", "jsonpath={.spec.projectRequestTemplate.name}")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check default project template",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking default project template: %v", err)
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "projectrequests.config.openshift.io", "cluster", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed project request configuration"
	}

	templateName := strings.TrimSpace(out)
	if templateName == "" {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"No custom default project template is configured",
			healthcheck.ResultKeyRecommended,
		)
		result = result.WithDetail(detailedOut)
		return result, nil
	}

	// Check if the template exists
	templateOut, err := utils.RunCommand("oc", "get", "template", templateName, "-n", "openshift-config")
	if err != nil {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			fmt.Sprintf("Default project template '%s' is configured but not found", templateName),
			healthcheck.ResultKeyRequired,
		)
		result = result.WithDetail(fmt.Sprintf("%s\n\n%s", detailedOut, err.Error()))
		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		fmt.Sprintf("Custom default project template '%s' is configured", templateName),
		healthcheck.ResultKeyNoChange,
	)
	result = result.WithDetail(fmt.Sprintf("%s\n\nTemplate:\n%s", detailedOut, templateOut))
	return result, nil
}

// DefaultNodeSelectorCheck checks if a default node selector is configured
type DefaultNodeSelectorCheck struct {
	healthcheck.BaseCheck
}

// NewDefaultNodeSelectorCheck creates a new default node selector check
func NewDefaultNodeSelectorCheck() *DefaultNodeSelectorCheck {
	return &DefaultNodeSelectorCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"default-node-selector",
			"Default Node Selector",
			"Checks if a default node selector is configured",
			healthcheck.CategoryCluster,
		),
	}
}

// Run executes the health check
func (c *DefaultNodeSelectorCheck) Run() (healthcheck.Result, error) {
	// Check if a default node selector is configured
	out, err := utils.RunCommand("oc", "get", "scheduler", "cluster",
		"-o", `jsonpath={.spec.defaultNodeSelector}`)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check default node selector",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking default node selector: %v", err)
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "scheduler", "cluster", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed scheduler configuration"
	}

	nodeSelector := strings.TrimSpace(out)
	if nodeSelector == "" {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"No default node selector is configured",
			healthcheck.ResultKeyRecommended,
		)
		result = result.WithDetail(detailedOut)
		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		fmt.Sprintf("Default node selector is configured: %s", nodeSelector),
		healthcheck.ResultKeyNoChange,
	)
	result = result.WithDetail(detailedOut)
	return result, nil
}

// KubeadminUserCheck checks if the kubeadmin user still exists
type KubeadminUserCheck struct {
	healthcheck.BaseCheck
}

// NewKubeadminUserCheck creates a new kubeadmin user check
func NewKubeadminUserCheck() *KubeadminUserCheck {
	return &KubeadminUserCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"kubeadmin-user",
			"Kubeadmin User",
			"Checks if the kubeadmin user still exists",
			healthcheck.CategoryCluster,
		),
	}
}

// Run executes the health check
func (c *KubeadminUserCheck) Run() (healthcheck.Result, error) {
	// Check if the kubeadmin secret exists
	out, err := utils.RunCommand("oc", "get", "secret", "kubeadmin", "-n", "kube-system")

	// If the command returns an error, it likely means the secret doesn't exist
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return healthcheck.NewResult(
				c.ID(),
				healthcheck.StatusOK,
				"The kubeadmin user has been removed",
				healthcheck.ResultKeyNoChange,
			), nil
		}

		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check for kubeadmin user",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking for kubeadmin user: %v", err)
	}

	// If we got here, the kubeadmin secret exists
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusWarning,
		"The kubeadmin user still exists and should be removed for security reasons",
		healthcheck.ResultKeyRecommended,
	)
	result = result.WithDetail(out)
	return result, nil
}

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
			healthcheck.CategoryCluster,
		),
	}
}

// Run executes the health check
func (c *InfrastructureProviderCheck) Run() (healthcheck.Result, error) {
	// Get the infrastructure provider type
	out, err := utils.RunCommand("oc", "get", "infrastructure", "cluster",
		"-o", "jsonpath={.status.platform}")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check infrastructure provider",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking infrastructure provider: %v", err)
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "infrastructure", "cluster", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed infrastructure configuration"
	}

	providerType := strings.TrimSpace(out)
	if providerType == "" {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"No infrastructure provider type detected",
			healthcheck.ResultKeyRequired,
		)
		result = result.WithDetail(detailedOut)
		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		fmt.Sprintf("Infrastructure provider type: %s", providerType),
		healthcheck.ResultKeyNoChange,
	)
	result = result.WithDetail(detailedOut)
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
		result = result.WithDetail(detailedOut)
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
	result = result.WithDetail(fmt.Sprintf("Infrastructure Name: %s\n\n%s", infrastructureName, detailedOut))
	return result, nil
}
