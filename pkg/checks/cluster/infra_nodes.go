package cluster

import (
	"context"
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
			types.CategoryClusterConfig,
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
			types.StatusCritical,
			"Failed to get Kubernetes client",
			types.ResultKeyRequired,
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
			types.StatusCritical,
			"Failed to retrieve infrastructure nodes",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving infrastructure nodes: %v", err)
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "nodes", "-l", "node-role.kubernetes.io/infra=", "-o", "wide")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed infrastructure node information"
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Check if there are any infrastructure nodes
	infraNodeCount := len(nodes.Items)
	if infraNodeCount == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No dedicated infrastructure nodes found",
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Configure dedicated infrastructure nodes")
		result.AddRecommendation("Infrastructure nodes allow you to isolate infrastructure workloads to prevent incurring billing costs against subscription counts and to separate maintenance and management")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/solutions/5034771"))

		result.Detail = detailedOut
		return result, nil
	}

	// Check if there are enough infrastructure nodes (at least 3 recommended)
	if infraNodeCount < 3 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Only %d infrastructure node(s) found, at least 3 are recommended", infraNodeCount),
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("In a production deployment, it is recommended that you deploy at least three infrastructure nodes")
		result.AddRecommendation("Both OpenShift Logging and Red Hat OpenShift Service Mesh deploy Elasticsearch, which requires three instances to be installed on different nodes")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/machine_management/index#creating-infrastructure-machinesets", version))

		result.Detail = detailedOut
		return result, nil
	}

	// Check if the infrastructure nodes are properly tainted
	allTainted := true
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
			allTainted = false
			break
		}
	}

	if !allTainted {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Found %d infrastructure nodes but not all are properly tainted", infraNodeCount),
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Add appropriate taints to infrastructure nodes to prevent regular workloads from being scheduled on them")
		result.AddRecommendation("Use 'oc adm taint nodes <node-name> node-role.kubernetes.io/infra=:NoSchedule'")

		result.Detail = detailedOut
		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("Found %d properly configured infrastructure nodes", infraNodeCount),
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
