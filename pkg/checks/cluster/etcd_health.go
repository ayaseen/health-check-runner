package cluster

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

		// Add detailed information
		fullDetail := fmt.Sprintf("ETCD Operator Information:\n%s\n\nETCD Pods Information:\n%s\n\nETCD Performance Information:\n%s",
			detailedOut, etcdPodsOut, perfDetails)
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

		// Add detailed information
		fullDetail := fmt.Sprintf("ETCD Operator Information:\n%s\n\nETCD Pods Information:\n%s\n\nETCD Performance Information:\n%s",
			detailedOut, etcdPodsOut, perfDetails)
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
	result.Detail = fmt.Sprintf("ETCD Operator Information:\n%s\n\nETCD Pods Information:\n%s\n\nETCD Performance Information:\n%s",
		detailedOut, etcdPodsOut, perfDetails)
	return result, nil
}

// EtcdPerformanceResult contains the result of etcd performance checks
type EtcdPerformanceResult struct {
	Healthy bool     // Overall health assessment
	Issues  []string // List of identified issues
}

// checkEtcdPerformance checks etcd performance metrics
func checkEtcdPerformance() (EtcdPerformanceResult, string) {
	result := EtcdPerformanceResult{
		Healthy: true,
		Issues:  []string{},
	}

	var details strings.Builder
	details.WriteString("=== ETCD Performance Metrics ===\n\n")

	// Check for slow compaction
	compactionOut, err := utils.RunCommand("oc", "logs", "--tail=500", "-n", "openshift-etcd", "-l", "app=etcd", "-c", "etcd", "|", "grep", "-i", "finished scheduled compaction")
	if err == nil {
		details.WriteString("Compaction times:\n")
		details.WriteString(compactionOut)
		details.WriteString("\n")

		// Parse compaction times
		compactionPattern := regexp.MustCompile(`finished scheduled compaction.*took\s+(\d+\.\d+)(m?s)`)
		matches := compactionPattern.FindAllStringSubmatch(compactionOut, -1)

		slowCompactions := 0
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

				// Check if duration is above threshold (100ms for small clusters, 800ms for large)
				if duration > 800 {
					slowCompactions++
				}
			}
		}

		if slowCompactions > 0 {
			result.Healthy = false
			result.Issues = append(result.Issues, fmt.Sprintf("slow etcd compactions detected (%d instances)", slowCompactions))
			details.WriteString("WARNING: Slow compactions detected. This indicates disk performance issues.\n\n")
		} else {
			details.WriteString("Compaction times are within acceptable range.\n\n")
		}
	}

	// Check for "took too long" messages
	tookTooLongOut, err := utils.RunCommand("oc", "logs", "--tail=500", "-n", "openshift-etcd", "-l", "app=etcd", "-c", "etcd", "|", "grep", "-i", "took too long")
	if err == nil && tookTooLongOut != "" {
		details.WriteString("Slow apply entries:\n")
		details.WriteString(tookTooLongOut)
		details.WriteString("\n")

		result.Healthy = false
		result.Issues = append(result.Issues, "etcd entries taking too long to apply")
		details.WriteString("WARNING: Entries taking too long to apply. This indicates disk performance issues.\n\n")
	} else {
		details.WriteString("No 'took too long' messages found (good).\n\n")
	}

	// Check for heartbeat issues
	heartbeatOut, err := utils.RunCommand("oc", "logs", "--tail=500", "-n", "openshift-etcd", "-l", "app=etcd", "-c", "etcd", "|", "grep", "-i", "failed to send out heartbeat")
	if err == nil && heartbeatOut != "" {
		details.WriteString("Heartbeat issues:\n")
		details.WriteString(heartbeatOut)
		details.WriteString("\n")

		result.Healthy = false
		result.Issues = append(result.Issues, "etcd heartbeat issues detected")
		details.WriteString("WARNING: Heartbeat issues detected. This typically indicates resource constraints.\n\n")
	} else {
		details.WriteString("No heartbeat issues found (good).\n\n")
	}

	// Check for clock drift
	clockDriftOut, err := utils.RunCommand("oc", "logs", "--tail=500", "-n", "openshift-etcd", "-l", "app=etcd", "-c", "etcd", "|", "grep", "-i", "clock difference")
	if err == nil && clockDriftOut != "" {
		details.WriteString("Clock drift issues:\n")
		details.WriteString(clockDriftOut)
		details.WriteString("\n")

		result.Healthy = false
		result.Issues = append(result.Issues, "etcd clock drift detected")
		details.WriteString("WARNING: Clock drift detected between etcd members. Check if chrony is properly configured.\n\n")
	} else {
		details.WriteString("No clock drift issues found (good).\n\n")
	}

	// Check disk wal fsync times using etcd diagnostic
	// Since we don't have direct access to Prometheus in this context, we'll use the etcdctl command to get metrics
	walFsyncOut, err := utils.RunCommandWithRetry(3, 2*time.Second, "oc", "exec", "-n", "openshift-etcd", "etcd-0", "-c", "etcd", "--", "etcdctl", "--command-timeout=60s", "check", "perf")
	if err == nil {
		details.WriteString("ETCD Performance Diagnostic Results:\n")
		details.WriteString(walFsyncOut)
		details.WriteString("\n")

		// Check if the output contains warnings about high latency
		if strings.Contains(walFsyncOut, "FAIL") || strings.Contains(walFsyncOut, "WARNING") {
			result.Healthy = false
			result.Issues = append(result.Issues, "etcd performance check indicates high latency")
			details.WriteString("WARNING: The etcd performance check indicates high latency issues.\n\n")
		} else {
			details.WriteString("ETCD performance diagnostics show good results.\n\n")
		}
	} else {
		details.WriteString("Could not run etcd performance diagnostics. This is non-critical.\n\n")
	}

	// Check CPU usage on etcd nodes
	clientset, err := getKubernetesClientset()
	if err == nil {
		// Get etcd pods
		pods, err := clientset.CoreV1().Pods("openshift-etcd").List(context.TODO(), metav1.ListOptions{
			LabelSelector: "app=etcd",
		})

		if err == nil && len(pods.Items) > 0 {
			details.WriteString("ETCD Pod Resources:\n")

			for _, pod := range pods.Items {
				nodeName := pod.Spec.NodeName

				// Get node CPU usage
				nodeUsage, err := utils.RunCommand("oc", "adm", "top", "node", nodeName)
				if err == nil {
					details.WriteString(fmt.Sprintf("Node %s resource usage:\n%s\n", nodeName, nodeUsage))

					// Check for high CPU usage
					cpuPattern := regexp.MustCompile(`(\d+)%`)
					cpuMatches := cpuPattern.FindStringSubmatch(nodeUsage)

					if len(cpuMatches) >= 2 {
						cpuUsage, _ := strconv.Atoi(cpuMatches[1])
						if cpuUsage > 80 {
							result.Healthy = false
							result.Issues = append(result.Issues, fmt.Sprintf("high CPU usage (%d%%) on etcd node %s", cpuUsage, nodeName))
							details.WriteString(fmt.Sprintf("WARNING: High CPU usage detected on node %s.\n", nodeName))
						}
					}
				}
			}
		}
	}

	// Add explanation of good etcd performance characteristics
	details.WriteString("\n=== ETCD Performance Best Practices ===\n")
	details.WriteString("For optimal etcd performance:\n")
	details.WriteString("1. Use fast SSD or NVMe storage dedicated to etcd\n")
	details.WriteString("2. Ensure 99th percentile of fsync is below 10ms\n")
	details.WriteString("3. Network latency between etcd nodes should be below 2ms\n")
	details.WriteString("4. CPU should not be overcommitted on etcd nodes\n")
	details.WriteString("5. Sequential fsync IOPS should be at least 500 for medium/large clusters\n\n")

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
