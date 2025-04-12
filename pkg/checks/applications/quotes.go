package applications

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// ResourceQuotasCheck checks if resource quotas and limits are configured
type ResourceQuotasCheck struct {
	healthcheck.BaseCheck
}

// NewResourceQuotasCheck creates a new resource quotas check
func NewResourceQuotasCheck() *ResourceQuotasCheck {
	return &ResourceQuotasCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"resource-quotas",
			"Resource Quotas",
			"Checks if resource quotas and limits are configured",
			healthcheck.CategoryApplications,
		),
	}
}

// Run executes the health check
func (c *ResourceQuotasCheck) Run() (healthcheck.Result, error) {
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

	// Get all namespaces
	ctx := context.Background()
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to retrieve namespaces",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error retrieving namespaces: %v", err)
	}

	// Counters for namespaces with and without resource quotas/limits
	totalUserNamespaces := 0
	namespacesWithResourceQuotas := 0
	namespacesWithLimitRanges := 0
	namespacesWithBoth := 0

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
	var namespacesWithoutQuotas []string
	var namespacesWithoutLimits []string
	var namespacesWithoutBoth []string

	// Check each namespace for resource quotas and limit ranges
	for _, namespace := range namespaces.Items {
		// Skip system namespaces
		if skipNamespaces[namespace.Name] || strings.HasPrefix(namespace.Name, "openshift-") {
			continue
		}

		totalUserNamespaces++

		// Check for resource quotas
		resourceQuotas, err := clientset.CoreV1().ResourceQuotas(namespace.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Non-critical error, we can continue with other namespaces
			continue
		}

		hasResourceQuotas := len(resourceQuotas.Items) > 0

		// Check for limit ranges
		limitRanges, err := clientset.CoreV1().LimitRanges(namespace.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Non-critical error, we can continue with other namespaces
			continue
		}

		hasLimitRanges := len(limitRanges.Items) > 0

		// Update counters
		if hasResourceQuotas {
			namespacesWithResourceQuotas++
		} else {
			namespacesWithoutQuotas = append(namespacesWithoutQuotas, namespace.Name)
		}

		if hasLimitRanges {
			namespacesWithLimitRanges++
		} else {
			namespacesWithoutLimits = append(namespacesWithoutLimits, namespace.Name)
		}

		if hasResourceQuotas && hasLimitRanges {
			namespacesWithBoth++
		} else {
			namespacesWithoutBoth = append(namespacesWithoutBoth, namespace.Name)
		}
	}

	// If there are no user namespaces, return NotApplicable
	if totalUserNamespaces == 0 {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusNotApplicable,
			"No user namespaces found in the cluster",
			healthcheck.ResultKeyNotApplicable,
		), nil
	}

	// Calculate percentages
	quotasPercentage := float64(namespacesWithResourceQuotas) / float64(totalUserNamespaces) * 100
	limitsPercentage := float64(namespacesWithLimitRanges) / float64(totalUserNamespaces) * 100
	bothPercentage := float64(namespacesWithBoth) / float64(totalUserNamespaces) * 100

	// Prepare a detailed description of what resource quotas and limit ranges are
	quotasDescription := `
What are Resource Quotas and Limit Ranges?

Resource Quotas: Define the total amount of resources a namespace can use. They limit the total CPU, memory, and other resources that can be consumed by all pods in a namespace.

Limit Ranges: Define default resource limits and requests for containers in a namespace. They can also enforce minimum and maximum resource usage limits.

Benefits of using Resource Quotas and Limit Ranges:
- Prevent resource starvation by limiting the total resources a namespace can consume
- Ensure fair resource allocation across namespaces
- Protect against runaway applications that might consume all available resources
- Enforce resource constraints and prevent resource leaks
- Help with capacity planning and cost management
`

	// If all namespaces have both resource quotas and limit ranges, the check passes
	if namespacesWithBoth == totalUserNamespaces {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			fmt.Sprintf("All %d user namespaces have both resource quotas and limit ranges configured", totalUserNamespaces),
			healthcheck.ResultKeyNoChange,
		)
		result.Detail = quotasDescription
		return result, nil
	}

	// Create result based on the percentage of namespaces with resource quotas and limit ranges
	var status healthcheck.Status
	var resultKey healthcheck.ResultKey
	var message string

	// Determine result status based on percentage of namespaces with resource constraints
	if bothPercentage < 30 {
		// Critical if less than 30% of namespaces have both resource quotas and limit ranges
		status = healthcheck.StatusWarning
		resultKey = healthcheck.ResultKeyRecommended
		message = fmt.Sprintf("Only %.1f%% of user namespaces (%d out of %d) have both resource quotas and limit ranges configured",
			bothPercentage, namespacesWithBoth, totalUserNamespaces)
	} else if quotasPercentage < 50 || limitsPercentage < 50 {
		// Warning if less than 50% of namespaces have either resource quotas or limit ranges
		status = healthcheck.StatusWarning
		resultKey = healthcheck.ResultKeyRecommended
		message = fmt.Sprintf("Many user namespaces are missing resource constraints: %.1f%% have resource quotas, %.1f%% have limit ranges",
			quotasPercentage, limitsPercentage)
	} else {
		// Otherwise, just an advisory
		status = healthcheck.StatusWarning
		resultKey = healthcheck.ResultKeyAdvisory
		message = fmt.Sprintf("Some user namespaces are missing resource constraints: %d missing resource quotas, %d missing limit ranges",
			totalUserNamespaces-namespacesWithResourceQuotas, totalUserNamespaces-namespacesWithLimitRanges)
	}

	result := healthcheck.NewResult(
		c.ID(),
		status,
		message,
		resultKey,
	)

	result.AddRecommendation("Configure resource quotas and limit ranges for all user namespaces")
	result.AddRecommendation("Follow the Kubernetes documentation on resource quotas: https://kubernetes.io/docs/concepts/policy/resource-quotas/")
	result.AddRecommendation("Follow the Kubernetes documentation on limit ranges: https://kubernetes.io/docs/concepts/policy/limit-range/")

	// Add detailed information
	detail := fmt.Sprintf("Summary:\n"+
		"- Total user namespaces: %d\n"+
		"- Namespaces with resource quotas: %d (%.1f%%)\n"+
		"- Namespaces with limit ranges: %d (%.1f%%)\n"+
		"- Namespaces with both: %d (%.1f%%)\n\n"+
		"Namespaces without resource quotas:\n- %s\n\n"+
		"Namespaces without limit ranges:\n- %s\n\n"+
		"Namespaces without both:\n- %s\n\n%s",
		totalUserNamespaces,
		namespacesWithResourceQuotas, quotasPercentage,
		namespacesWithLimitRanges, limitsPercentage,
		namespacesWithBoth, bothPercentage,
		strings.Join(namespacesWithoutQuotas, "\n- "),
		strings.Join(namespacesWithoutLimits, "\n- "),
		strings.Join(namespacesWithoutBoth, "\n- "),
		quotasDescription)

	result.Detail = detail

	return result, nil
}
