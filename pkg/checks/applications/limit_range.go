package applications

import (
	"context"
	"fmt"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LimitRangeCheck checks if resource limits are configured
type LimitRangeCheck struct {
	healthcheck.BaseCheck
}

// NewLimitRangeCheck creates a new limit range check
func NewLimitRangeCheck() *LimitRangeCheck {
	return &LimitRangeCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"limit-range",
			"LimitRange Configuration",
			"Checks if LimitRange is configured in user namespaces",
			types.CategoryAppDev,
		),
	}
}

// Run executes the health check
func (c *LimitRangeCheck) Run() (healthcheck.Result, error) {
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

	// Get all namespaces
	ctx := context.Background()
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to retrieve namespaces",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving namespaces: %v", err)
	}

	// Counters for namespaces with and without limit ranges
	totalUserNamespaces := 0
	namespacesWithLimitRanges := 0

	// Namespaces to skip (system namespaces)
	skipNamespaces := map[string]bool{
		"default":             true,
		"kube-system":         true,
		"kube-public":         true,
		"kube-node-lease":     true,
		"openshift":           true,
		"openshift-etcd":      true,
		"openshift-apiserver": true,
	}

	// Lists to collect details
	var namespacesWithoutLimitRanges []string

	// Check each namespace for limit ranges
	for _, namespace := range namespaces.Items {
		// Skip system namespaces
		if skipNamespaces[namespace.Name] || strings.HasPrefix(namespace.Name, "openshift-") {
			continue
		}

		totalUserNamespaces++

		// Check for limit ranges
		limitRanges, err := clientset.CoreV1().LimitRanges(namespace.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Non-critical error, we can continue with other namespaces
			continue
		}

		if len(limitRanges.Items) > 0 {
			namespacesWithLimitRanges++
		} else {
			namespacesWithoutLimitRanges = append(namespacesWithoutLimitRanges, namespace.Name)
		}
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "limitranges", "--all-namespaces")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed LimitRange information"
	}

	// If there are no user namespaces, return NotApplicable
	if totalUserNamespaces == 0 {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"No user namespaces found in the cluster",
			types.ResultKeyNotApplicable,
		), nil
	}

	// Calculate percentage
	limitRangePercentage := float64(namespacesWithLimitRanges) / float64(totalUserNamespaces) * 100

	// Generate the appropriate result based on findings
	if namespacesWithLimitRanges == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"No namespaces have LimitRange configured",
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure LimitRange resources in your namespaces to control resource usage")
		result.AddRecommendation("Follow best practices for resource management: https://kubernetes.io/docs/concepts/policy/limit-range/")
		result.Detail = detailedOut
		return result, nil
	} else if limitRangePercentage < 50 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Only %.1f%% of user namespaces (%d out of %d) have LimitRange configured",
				limitRangePercentage, namespacesWithLimitRanges, totalUserNamespaces),
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure LimitRange resources in all namespaces to control resource usage")
		result.AddRecommendation("Set up a default project template including LimitRange")
		result.Detail = fmt.Sprintf("Namespaces without LimitRange:\n- %s\n\n%s",
			strings.Join(namespacesWithoutLimitRanges, "\n- "), detailedOut)
		return result, nil
	} else if limitRangePercentage < 100 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("%.1f%% of user namespaces (%d out of %d) have LimitRange configured",
				limitRangePercentage, namespacesWithLimitRanges, totalUserNamespaces),
			types.ResultKeyAdvisory,
		)
		result.AddRecommendation("Configure LimitRange resources in all remaining namespaces")
		result.Detail = fmt.Sprintf("Namespaces without LimitRange:\n- %s\n\n%s",
			strings.Join(namespacesWithoutLimitRanges, "\n- "), detailedOut)
		return result, nil
	}

	// All namespaces have LimitRange
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("All %d user namespaces have LimitRange configured", totalUserNamespaces),
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
