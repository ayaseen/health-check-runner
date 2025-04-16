/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for etcd cluster health. It:

- Examines the status of the etcd cluster and operator
- Checks for signs of performance issues or degradation
- Analyzes etcd logs for common problem patterns
- Provides detailed diagnostics about etcd health
- Recommends actions to address etcd issues

This check helps ensure the stability of the control plane by monitoring the critical etcd database component.
*/

package cluster

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
	"k8s.io/client-go/kubernetes"
	"regexp"
	"strconv"
	"strings"
)

// EtcdHealthCheck checks the health of the etcd cluster and its performance
type EtcdHealthCheck struct {
	healthcheck.BaseCheck
}

// NewEtcdHealthCheck creates a new etcd health check
func NewEtcdHealthCheck() *EtcdHealthCheck {
	return &EtcdHealthCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"etcd-health",
			"ETCD Health",
			"Checks the health of the etcd cluster and performance metrics",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *EtcdHealthCheck) Run() (healthcheck.Result, error) {
	// Check for degraded etcd operator status
	out, err := utils.RunCommand("oc", "get", "co", "etcd", "-o", "jsonpath={.status.conditions[?(@.type==\"Degraded\")].status}")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get etcd operator status",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting etcd operator status: %v", err)
	}

	degraded := strings.TrimSpace(out)

	// Get detailed etcd operator information
	detailedOut, err := utils.RunCommand("oc", "get", "co", "etcd", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed etcd operator information"
	}

	// Check for available etcd operator status
	availableOut, err := utils.RunCommand("oc", "get", "co", "etcd", "-o", "jsonpath={.status.conditions[?(@.type==\"Available\")].status}")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get etcd operator availability status",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting etcd operator availability status: %v", err)
	}

	available := strings.TrimSpace(availableOut)

	// Check etcd cluster status by looking at specific metrics
	// For a simplified check, we'll just examine if the etcd service is running and if the pods are healthy
	etcdPodsOut, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-etcd", "-l", "app=etcd")
	if err != nil {
		// Non-critical error, we can continue with the other information
		etcdPodsOut = "Failed to get etcd pods information"
	}

	// Check if all etcd pods are running
	allPodsRunning := true
	if !strings.Contains(etcdPodsOut, "Running") {
		allPodsRunning = false
	}

	// Get etcd performance metrics
	perfResult, perfDetails := checkEtcdPerformance()

	// Create the exact format for the detail output with proper spacing
	var formattedDetailedOut string
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailedOut = fmt.Sprintf("ETCD Operator Information:\n[source, yaml]\n----\n%s\n----\n\n", detailedOut)
	} else {
		formattedDetailedOut = "ETCD Operator Information: No information available\n\n"
	}

	var formattedPodsOut string
	if strings.TrimSpace(etcdPodsOut) != "" {
		formattedPodsOut = fmt.Sprintf("ETCD Pods Information:\n[source, bash]\n----\n%s\n----\n\n", etcdPodsOut)
	} else {
		formattedPodsOut = "ETCD Pods Information: No information available\n\n"
	}

	// If etcd is degraded or not available, the check fails
	if degraded == "True" || available != "True" || !allPodsRunning {
		status := types.StatusCritical
		resultKey := types.ResultKeyRequired

		// Create detailed message based on the issues found
		var issues []string
		if degraded == "True" {
			issues = append(issues, "etcd operator is degraded")
		}
		if available != "True" {
			issues = append(issues, "etcd operator is not available")
		}
		if !allPodsRunning {
			issues = append(issues, "not all etcd pods are running")
		}

		// Add performance issues if any
		if !perfResult.Healthy {
			issues = append(issues, perfResult.Issues...)
			// If we only have performance issues, downgrade to warning
			if len(issues) == len(perfResult.Issues) {
				status = types.StatusWarning
				resultKey = types.ResultKeyRecommended
			}
		}

		// Create result with etcd issues
		result := healthcheck.NewResult(
			c.ID(),
			status,
			fmt.Sprintf("ETCD cluster has issues: %s", strings.Join(issues, ", ")),
			resultKey,
		)

		result.AddRecommendation("Investigate etcd issues using 'oc get co etcd -o yaml'")
		result.AddRecommendation("Check etcd pod logs using 'oc logs -n openshift-etcd etcd-<node-name>'")
		result.AddRecommendation("Consult the documentation at https://docs.openshift.com/container-platform/latest/scalability_and_performance/recommended-host-practices.html#recommended-etcd-practices_recommended-host-practices")

		// Add performance-specific recommendations
		if !perfResult.Healthy {
			result.AddRecommendation("Ensure etcd is running on fast SSD or NVMe storage for optimal performance")
			result.AddRecommendation("Make sure that nodes have sufficient resources and are not over-committed")
			result.AddRecommendation("Check network latency between etcd nodes (should be below 2ms for optimal performance)")
		}

		// Combine all the sections with proper formatting
		fullDetail := formattedDetailedOut + formattedPodsOut + "ETCD Performance Information:\n\n" + perfDetails

		result.Detail = fullDetail
		return result, nil
	}

	// Check performance issues separately if etcd is otherwise healthy
	if !perfResult.Healthy {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("ETCD cluster has performance issues: %s", strings.Join(perfResult.Issues, ", ")),
			types.ResultKeyRecommended,
		)

		result.AddRecommendation("Ensure etcd is running on fast SSD or NVMe storage for optimal performance")
		result.AddRecommendation("Check that fsync times are below 10ms for good performance")
		result.AddRecommendation("Make sure that nodes have sufficient resources and are not over-committed")
		result.AddRecommendation("Check network latency between etcd nodes (should be below 2ms for optimal performance)")
		result.AddRecommendation("Consult the documentation at https://docs.openshift.com/container-platform/latest/scalability_and_performance/recommended-host-practices.html#recommended-etcd-practices_recommended-host-practices")

		// Combine all the sections with proper formatting
		fullDetail := formattedDetailedOut + formattedPodsOut + "ETCD Performance Information:\n\n" + perfDetails

		result.Detail = fullDetail
		return result, nil
	}

	// If everything looks good, return OK
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"ETCD cluster is healthy and performing well",
		types.ResultKeyNoChange,
	)

	// Combine all the sections with proper formatting
	fullDetail := formattedDetailedOut + formattedPodsOut + "ETCD Performance Information:\n\n" + perfDetails

	result.Detail = fullDetail
	return result, nil
}

// EtcdPerformanceResult contains the result of etcd performance checks
type EtcdPerformanceResult struct {
	Healthy bool     // Overall health assessment
	Issues  []string // List of identified issues
}

// Improved checkEtcdPerformance function with correct AsciiDoc formatting
func checkEtcdPerformance() (EtcdPerformanceResult, string) {
	result := EtcdPerformanceResult{
		Healthy: true,
		Issues:  []string{},
	}

	var details strings.Builder
	details.WriteString("=== ETCD Performance Metrics ===\n\n")

	// Check for slow compaction
	compactionOut, err := utils.RunCommand("oc", "logs", "--tail=500", "-n", "openshift-etcd", "-l", "app=etcd", "-c", "etcd", "|", "grep", "-i", "finished scheduled compaction")
	if err == nil && compactionOut != "" {
		details.WriteString("Compaction times:\n")
		details.WriteString("[source, text]\n----\n")
		details.WriteString(compactionOut)
		details.WriteString("\n----\n\n")

		// Parse compaction times
		compactionPattern := regexp.MustCompile(`finished scheduled compaction.*took\s+(\d+\.\d+)(m?s)`)
		matches := compactionPattern.FindAllStringSubmatch(compactionOut, -1)

		slowCompactions := 0
		verySlowCompactions := 0
		var compactionTimes []float64

		for _, match := range matches {
			if len(match) >= 3 {
				duration, _ := strconv.ParseFloat(match[1], 64)
				unit := match[2]

				// Convert to milliseconds if necessary
				if unit == "ms" {
					// Already in ms
				} else {
					// Assuming it's in seconds
					duration *= 1000
				}

				compactionTimes = append(compactionTimes, duration)

				// Check thresholds
				if duration > 800 {
					verySlowCompactions++
				} else if duration > 100 {
					slowCompactions++
				}
			}
		}

		// Calculate average compaction time
		var avgCompactionTime float64
		if len(compactionTimes) > 0 {
			sum := 0.0
			for _, time := range compactionTimes {
				sum += time
			}
			avgCompactionTime = sum / float64(len(compactionTimes))
		}

		details.WriteString("Compaction statistics:\n")
		details.WriteString(fmt.Sprintf("- Total compactions analyzed: %d\n", len(compactionTimes)))
		details.WriteString(fmt.Sprintf("- Average compaction time: %.2f ms\n", avgCompactionTime))
		details.WriteString(fmt.Sprintf("- Slow compactions (> 100ms): %d\n", slowCompactions))
		details.WriteString(fmt.Sprintf("- Very slow compactions (> 800ms): %d\n\n", verySlowCompactions))

		// For small clusters, compaction should be below 10ms
		// For large clusters, compaction should be below 100ms
		if verySlowCompactions > 0 {
			result.Healthy = false
			result.Issues = append(result.Issues, fmt.Sprintf("very slow etcd compactions detected (%d instances > 800ms)", verySlowCompactions))
			details.WriteString("WARNING: Very slow compactions detected. This indicates severe disk performance issues.\n\n")
		} else if slowCompactions > 3 { // Allow a few slow compactions as normal
			result.Healthy = false
			result.Issues = append(result.Issues, fmt.Sprintf("slow etcd compactions detected (%d instances > 100ms)", slowCompactions))
			details.WriteString("WARNING: Multiple slow compactions detected. This may indicate disk performance issues.\n\n")
		}

		if avgCompactionTime > 50 { // Average should be much lower than the occasional spikes
			result.Healthy = false
			result.Issues = append(result.Issues, fmt.Sprintf("high average etcd compaction time (%.1fms)", avgCompactionTime))
			details.WriteString("WARNING: Average compaction time is higher than expected.\n\n")
		}
	}

	// Check for "took too long" issues
	tookTooLongOut, err := utils.RunCommand("oc", "logs", "--tail=500", "-n", "openshift-etcd", "-l", "app=etcd", "-c", "etcd", "|", "grep", "-i", "took too long")
	if err == nil && tookTooLongOut != "" {
		details.WriteString("Apply entries performance logs:\n")
		details.WriteString("[source, text]\n----\n")
		details.WriteString(tookTooLongOut)
		details.WriteString("\n----\n\n")

		// Count occurrences
		lines := strings.Split(tookTooLongOut, "\n")
		numOccurrences := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				numOccurrences++
			}
		}

		// Look for time patterns like "(123.45ms)" to analyze severity
		timePattern := regexp.MustCompile(`took too long \(([0-9.]+)ms\)`)
		matches := timePattern.FindAllStringSubmatch(tookTooLongOut, -1)

		var highDelays int
		var veryHighDelays int
		for _, match := range matches {
			if len(match) >= 2 {
				delay, _ := strconv.ParseFloat(match[1], 64)
				if delay > 500 {
					veryHighDelays++
				} else if delay > 100 {
					highDelays++
				}
			}
		}

		details.WriteString("Apply entries statistics:\n")
		details.WriteString(fmt.Sprintf("- Total 'took too long' messages: %d\n", numOccurrences))
		details.WriteString(fmt.Sprintf("- High delay entries (> 100ms): %d\n", highDelays))
		details.WriteString(fmt.Sprintf("- Very high delay entries (> 500ms): %d\n\n", veryHighDelays))

		// Evaluate the severity
		if numOccurrences > 20 || veryHighDelays > 5 {
			result.Healthy = false
			result.Issues = append(result.Issues, fmt.Sprintf("excessive delays applying etcd entries (%d occurrences)", numOccurrences))
			details.WriteString("CRITICAL: Frequent delays applying etcd entries. This indicates serious disk performance issues.\n\n")
		} else if numOccurrences > 5 || highDelays > 2 {
			result.Healthy = false
			result.Issues = append(result.Issues, fmt.Sprintf("delays applying etcd entries (%d occurrences)", numOccurrences))
			details.WriteString("WARNING: Delays applying etcd entries. This indicates disk performance issues.\n\n")
		}
	} else {
		details.WriteString("No 'took too long' messages found in recent logs (good).\n\n")
	}

	// Check for heartbeat issues
	heartbeatOut, err := utils.RunCommand("oc", "logs", "--tail=500", "-n", "openshift-etcd", "-l", "app=etcd", "-c", "etcd", "|", "grep", "-i", "failed to send out heartbeat")
	if err == nil && heartbeatOut != "" {
		details.WriteString("Heartbeat issues:\n")
		details.WriteString("[source, text]\n----\n")
		details.WriteString(heartbeatOut)
		details.WriteString("\n----\n\n")

		// Count occurrences
		lines := strings.Split(heartbeatOut, "\n")
		numOccurrences := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				numOccurrences++
			}
		}

		details.WriteString("Heartbeat statistics:\n")
		details.WriteString(fmt.Sprintf("- Total heartbeat failure messages: %d\n\n", numOccurrences))

		// Evaluate the severity
		if numOccurrences > 20 {
			result.Healthy = false
			result.Issues = append(result.Issues, fmt.Sprintf("frequent heartbeat failures (%d occurrences)", numOccurrences))
			details.WriteString("CRITICAL: Frequent heartbeat failures. This indicates resource constraints or disk performance issues.\n\n")
		} else if numOccurrences > 5 {
			result.Healthy = false
			result.Issues = append(result.Issues, fmt.Sprintf("heartbeat failures (%d occurrences)", numOccurrences))
			details.WriteString("WARNING: Heartbeat failures detected. This indicates resource constraints or disk performance issues.\n\n")
		}
	} else {
		details.WriteString("No heartbeat issues found in recent logs (good).\n\n")
	}

	// Check for clock drift issues
	clockDriftOut, err := utils.RunCommand("oc", "logs", "--tail=500", "-n", "openshift-etcd", "-l", "app=etcd", "-c", "etcd", "|", "grep", "-i", "clock difference")
	if err == nil && clockDriftOut != "" {
		details.WriteString("Clock drift logs:\n")
		details.WriteString("[source, text]\n----\n")
		details.WriteString(clockDriftOut)
		details.WriteString("\n----\n\n")

		// Evaluate the severity
		result.Healthy = false
		result.Issues = append(result.Issues, "clock drift between etcd nodes")
		details.WriteString("WARNING: Clock drift detected between etcd nodes. Verify chrony configuration.\n\n")
	} else {
		details.WriteString("No clock drift issues found in recent logs (good).\n\n")
	}

	// Add explanation of good etcd performance characteristics
	details.WriteString("=== ETCD Performance Best Practices ===\n")
	details.WriteString("For optimal etcd performance:\n")
	details.WriteString("1. Use fast SSD or NVMe storage dedicated to etcd\n")
	details.WriteString("2. Ensure 99th percentile of fsync is below 10ms\n")
	details.WriteString("3. Network latency between etcd nodes should be below 2ms\n")
	details.WriteString("4. CPU should not be overcommitted on etcd nodes\n")
	details.WriteString("5. Sequential fsync IOPS should be at least 500 for medium/large clusters\n")

	return result, details.String()
}

// getKubernetesClientset returns a Kubernetes clientset
func getKubernetesClientset() (*kubernetes.Clientset, error) {
	config, err := utils.GetClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return clientset, nil
}
