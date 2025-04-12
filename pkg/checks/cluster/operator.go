package cluster

import (
	"context"
	"fmt"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// ClusterOperatorsCheck checks if all cluster operators are available
type ClusterOperatorsCheck struct {
	healthcheck.BaseCheck
}

// NewClusterOperatorsCheck creates a new cluster operators check
func NewClusterOperatorsCheck() *ClusterOperatorsCheck {
	return &ClusterOperatorsCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"cluster-operators",
			"Cluster Operators",
			"Checks if all cluster operators are available",
			healthcheck.CategoryCluster,
		),
	}
}

// Run executes the health check
func (c *ClusterOperatorsCheck) Run() (healthcheck.Result, error) {
	// Get Kubernetes config
	config, err := utils.GetClusterConfig()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get cluster config",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting cluster config: %v", err)
	}

	// Create OpenShift client
	client, err := versioned.NewForConfig(config)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to create OpenShift client",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error creating OpenShift client: %v", err)
	}

	// Get the list of cluster operators
	ctx := context.Background()
	cos, err := client.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to retrieve cluster operators",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error retrieving cluster operators: %v", err)
	}

	// Check if all cluster operators are available
	allAvailable := true
	var unavailableOps []string

	for _, co := range cos.Items {
		available := false

		for _, condition := range co.Status.Conditions {
			if condition.Type == configv1.OperatorAvailable && condition.Status == configv1.ConditionTrue {
				available = true
				break
			}
		}

		if !available {
			allAvailable = false
			unavailableOps = append(unavailableOps, co.Name)
		}
	}

	// Get the output of 'oc get co' for detailed information
	detailedOut, err := utils.RunCommand("oc", "get", "co")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed operator status"
	}

	if allAvailable {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			"All cluster operators are available",
			healthcheck.ResultKeyNoChange,
		)
		result = result.WithDetail(detailedOut)
		return result, nil
	}

	// Create result with unavailable operators information
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusCritical,
		fmt.Sprintf("Some cluster operators are not available: %s", strings.Join(unavailableOps, ", ")),
		healthcheck.ResultKeyRequired,
	)

	result.AddRecommendation("Investigate why the operators are not available")
	result.AddRecommendation("Check operator logs using 'oc logs deployment/<operator-name> -n <operator-namespace>'")
	result.AddRecommendation("Consult the OpenShift documentation or Red Hat support")

	result = result.WithDetail(detailedOut)
	return result, nil
}
