/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for resource quotas and limits. It:

- Examines namespaces for configured resource quotas and limit ranges
- Identifies namespaces lacking proper resource constraints
- Calculates the percentage of namespaces with appropriate configurations
- Provides detailed recommendations for resource management
- Explains the purpose and benefits of resource quotas and limit ranges

These checks help ensure fair resource allocation and prevent resource starvation in multi-tenant OpenShift environments.
*/

package applications

import (
	"context"
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
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
			types.CategoryApplications,
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

	// Format the detailed output with proper AsciiDoc formatting
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Resource Quotas and Limits Analysis ===\n\n")

	// Add namespace statistics with proper formatting
	formattedDetailOut.WriteString("Namespace Statistics:\n")
	formattedDetailOut.WriteString(fmt.Sprintf("- Total User Namespaces: %d\n", totalUserNamespaces))
	formattedDetailOut.WriteString(fmt.Sprintf("- Namespaces with Resource Quotas: %d", namespacesWithResourceQuotas))

	if totalUserNamespaces > 0 {
		quotasPercentage := float64(namespacesWithResourceQuotas) / float64(totalUserNamespaces) * 100
		formattedDetailOut.WriteString(fmt.Sprintf(" (%.1f%%)\n", quotasPercentage))
	} else {
		formattedDetailOut.WriteString(" (N/A)\n")
	}

	formattedDetailOut.WriteString(fmt.Sprintf("- Namespaces with Limit Ranges: %d", namespacesWithLimitRanges))

	if totalUserNamespaces > 0 {
		limitsPercentage := float64(namespacesWithLimitRanges) / float64(totalUserNamespaces) * 100
		formattedDetailOut.WriteString(fmt.Sprintf(" (%.1f%%)\n", limitsPercentage))
	} else {
		formattedDetailOut.WriteString(" (N/A)\n")
	}

	formattedDetailOut.WriteString(fmt.Sprintf("- Namespaces with Both: %d", namespacesWithBoth))

	if totalUserNamespaces > 0 {
		bothPercentage := float64(namespacesWithBoth) / float64(totalUserNamespaces) * 100
		formattedDetailOut.WriteString(fmt.Sprintf(" (%.1f%%)\n\n", bothPercentage))
	} else {
		formattedDetailOut.WriteString(" (N/A)\n\n")
	}

	// Add namespaces without quotas information with proper formatting
	if len(namespacesWithoutQuotas) > 0 {
		formattedDetailOut.WriteString("Namespaces Without Resource Quotas:\n[source, text]\n----\n")
		for _, ns := range namespacesWithoutQuotas {
			formattedDetailOut.WriteString(fmt.Sprintf("- %s\n", ns))
		}
		formattedDetailOut.WriteString("----\n\n")
	} else if totalUserNamespaces > 0 {
		formattedDetailOut.WriteString("Namespaces Without Resource Quotas: None (all namespaces have resource quotas)\n\n")
	}

	// Add namespaces without limits information with proper formatting
	if len(namespacesWithoutLimits) > 0 {
		formattedDetailOut.WriteString("Namespaces Without Limit Ranges:\n[source, text]\n----\n")
		for _, ns := range namespacesWithoutLimits {
			formattedDetailOut.WriteString(fmt.Sprintf("- %s\n", ns))
		}
		formattedDetailOut.WriteString("----\n\n")
	} else if totalUserNamespaces > 0 {
		formattedDetailOut.WriteString("Namespaces Without Limit Ranges: None (all namespaces have limit ranges)\n\n")
	}

	// Add namespaces without both information with proper formatting
	if len(namespacesWithoutBoth) > 0 {
		formattedDetailOut.WriteString("Namespaces Without Both Resource Quotas and Limit Ranges:\n[source, text]\n----\n")
		for _, ns := range namespacesWithoutBoth {
			formattedDetailOut.WriteString(fmt.Sprintf("- %s\n", ns))
		}
		formattedDetailOut.WriteString("----\n\n")
	} else if totalUserNamespaces > 0 {
		formattedDetailOut.WriteString("Namespaces Without Both: None (all namespaces have both resource quotas and limit ranges)\n\n")
	}

	// Add resource quotas and limit ranges documentation
	formattedDetailOut.WriteString("=== Resource Management Information ===\n\n")
	formattedDetailOut.WriteString("What are Resource Quotas and Limit Ranges?\n\n")
	formattedDetailOut.WriteString("Resource Quotas: Define the total amount of resources a namespace can use. They limit the total CPU, memory, and other resources that can be consumed by all pods in a namespace.\n\n")
	formattedDetailOut.WriteString("Limit Ranges: Define default resource limits and requests for containers in a namespace. They can also enforce minimum and maximum resource usage limits.\n\n")
	formattedDetailOut.WriteString("Benefits of using Resource Quotas and Limit Ranges:\n")
	formattedDetailOut.WriteString("- Prevent resource starvation by limiting the total resources a namespace can consume\n")
	formattedDetailOut.WriteString("- Ensure fair resource allocation across namespaces\n")
	formattedDetailOut.WriteString("- Protect against runaway applications that might consume all available resources\n")
	formattedDetailOut.WriteString("- Enforce resource constraints and prevent resource leaks\n")
	formattedDetailOut.WriteString("- Help with capacity planning and cost management\n\n")

	// If there are no user namespaces, return NotApplicable
	if totalUserNamespaces == 0 {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"No user namespaces found in the cluster",
			types.ResultKeyNotApplicable,
		), nil
	}

	// If all namespaces have both resource quotas and limit ranges, the check passes
	if namespacesWithBoth == totalUserNamespaces {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			fmt.Sprintf("All %d user namespaces have both resource quotas and limit ranges configured", totalUserNamespaces),
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Calculate percentages
	quotasPercentage := float64(namespacesWithResourceQuotas) / float64(totalUserNamespaces) * 100
	limitsPercentage := float64(namespacesWithLimitRanges) / float64(totalUserNamespaces) * 100
	bothPercentage := float64(namespacesWithBoth) / float64(totalUserNamespaces) * 100

	// Create result based on the percentage of namespaces with resource constraints
	var status types.Status
	var resultKey types.ResultKey
	var message string

	// Determine result status based on percentage of namespaces with resource constraints
	if bothPercentage < 30 {
		// Critical if less than 30% of namespaces have both resource quotas and limit ranges
		status = types.StatusWarning
		resultKey = types.ResultKeyRecommended
		message = fmt.Sprintf("Only %.1f%% of user namespaces (%d out of %d) have both resource quotas and limit ranges configured",
			bothPercentage, namespacesWithBoth, totalUserNamespaces)
	} else if quotasPercentage < 50 || limitsPercentage < 50 {
		// Warning if less than 50% of namespaces have either resource quotas or limit ranges
		status = types.StatusWarning
		resultKey = types.ResultKeyRecommended
		message = fmt.Sprintf("Many user namespaces are missing resource constraints: %.1f%% have resource quotas, %.1f%% have limit ranges",
			quotasPercentage, limitsPercentage)
	} else {
		// Otherwise, just an advisory
		status = types.StatusWarning
		resultKey = types.ResultKeyAdvisory
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

	result.Detail = formattedDetailOut.String()
	return result, nil
}
