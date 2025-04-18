/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-18

This file implements health checks for user workload performance in non-OpenShift namespaces. It:

- Identifies non-OpenShift namespaces containing user workloads
- Gathers CPU and memory metrics for user workloads within those namespaces
- Analyzes resource utilization, requests, and limits for each workload
- Identifies workloads with high resource consumption or inefficient resource allocation
- Provides recommendations for optimizing resource requests and limits
- Helps maintain optimal performance for user applications

This check helps administrators understand resource consumption patterns and optimize application performance.
*/

package performance

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// WorkloadMetric represents resource metrics for a single workload
type WorkloadMetric struct {
	Namespace  string
	Workload   string
	Value      float64
	HumanValue string
}

// NamespaceWorkloadMetrics holds workload metrics for a namespace
type NamespaceWorkloadMetrics struct {
	Namespace string
	Pods      []WorkloadMetric
	CPUUsage  []WorkloadMetric
	CPUReq    []WorkloadMetric
	CPULim    []WorkloadMetric
	MemUsage  []WorkloadMetric
	MemReq    []WorkloadMetric
	MemLim    []WorkloadMetric
	// Derived metrics
	CPUUtilization []WorkloadMetric // Usage as percentage of request
	MemUtilization []WorkloadMetric // Usage as percentage of request
}

// UserWorkloadPerformanceCheck checks the performance of user workloads in non-OpenShift namespaces
type UserWorkloadPerformanceCheck struct {
	healthcheck.BaseCheck
	// Number of top workloads to show
	TopCount int
	// Thresholds for CPU utilization warnings (as percentage of requests)
	CPUUtilizationWarning  float64
	CPUUtilizationCritical float64
	// Thresholds for memory utilization warnings (as percentage of requests)
	MemUtilizationWarning  float64
	MemUtilizationCritical float64
}

// NewUserWorkloadPerformanceCheck creates a new user workload performance check
func NewUserWorkloadPerformanceCheck() *UserWorkloadPerformanceCheck {
	return &UserWorkloadPerformanceCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"user-workload-performance",
			"User Workload Performance",
			"Checks performance metrics for user workloads in non-OpenShift namespaces",
			types.CategoryPerformance,
		),
		TopCount:               5,
		CPUUtilizationWarning:  80.0,  // 80% of requested CPU
		CPUUtilizationCritical: 100.0, // 100% of requested CPU
		MemUtilizationWarning:  80.0,  // 80% of requested memory
		MemUtilizationCritical: 90.0,  // 90% of requested memory
	}
}

// Run executes the health check
func (c *UserWorkloadPerformanceCheck) Run() (healthcheck.Result, error) {
	// Create a timeout context to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Get all user namespaces (non-OpenShift, non-kube namespaces)
	userNamespaces, err := c.getUserNamespaces(ctx)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get user namespaces",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting user namespaces: %v", err)
	}

	if len(userNamespaces) == 0 {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"No user namespaces found",
			types.ResultKeyNotApplicable,
		), nil
	}

	// Get token for authentication
	token, err := utils.RunCommand("oc", "whoami", "-t")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get authentication token",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting authentication token: %v", err)
	}
	token = strings.TrimSpace(token)

	// Get thanos querier host
	host, err := utils.RunCommand("oc", "-n", "openshift-monitoring", "get", "route", "thanos-querier", "-o", "jsonpath={.status.ingress[0].host}")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get Thanos Querier host",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting Thanos Querier host: %v", err)
	}
	host = strings.TrimSpace(host)

	// Collect metrics for each namespace
	allNamespaceMetrics := make(map[string]*NamespaceWorkloadMetrics)

	for _, ns := range userNamespaces {
		metrics, err := c.collectNamespaceWorkloadMetrics(ctx, ns, host, token)
		if err != nil {
			// Log the error but continue with other namespaces
			fmt.Printf("Warning: Failed to collect metrics for namespace %s: %v\n", ns, err)
			continue
		}

		// Skip namespaces with no workloads
		if len(metrics.CPUUsage) == 0 && len(metrics.MemUsage) == 0 {
			continue
		}

		allNamespaceMetrics[ns] = metrics
	}

	if len(allNamespaceMetrics) == 0 {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"No user workloads found in user namespaces",
			types.ResultKeyNotApplicable,
		), nil
	}

	// Identify problematic workloads
	highCPUWorkloads, highMemWorkloads := c.identifyProblematicWorkloads(allNamespaceMetrics)

	// Build detailed output with proper AsciiDoc formatting
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== User Workload Performance Analysis ===\n\n")
	formattedDetailOut.WriteString(fmt.Sprintf("Analysis of user workloads across %d user namespaces.\n\n", len(allNamespaceMetrics)))

	// Add summary of analyzed namespaces
	formattedDetailOut.WriteString("== User Namespaces Analyzed ==\n\n")
	formattedDetailOut.WriteString("The following user namespaces were analyzed:\n\n")
	for ns := range allNamespaceMetrics {
		formattedDetailOut.WriteString(fmt.Sprintf("* %s\n", ns))
	}
	formattedDetailOut.WriteString("\n")

	// Add problematic workloads section if any were found
	if len(highCPUWorkloads) > 0 || len(highMemWorkloads) > 0 {
		formattedDetailOut.WriteString("== Problematic Workloads ==\n\n")

		if len(highCPUWorkloads) > 0 {
			formattedDetailOut.WriteString("=== High CPU Utilization Workloads ===\n\n")
			formattedDetailOut.WriteString("{set:cellbgcolor!}\n") // Reset any cell background color
			formattedDetailOut.WriteString("[cols=\"1,1,1,1,1\", options=\"header\"]\n|===\n")
			formattedDetailOut.WriteString("|Namespace|Workload|CPU Usage|CPU Request|Utilization\n")

			for _, wl := range highCPUWorkloads {
				formattedDetailOut.WriteString(fmt.Sprintf("|%s|%s|%s|%s|%.2f%%\n",
					wl.Namespace,
					wl.Workload,
					findHumanValue(allNamespaceMetrics[wl.Namespace].CPUUsage, wl.Workload),
					findHumanValue(allNamespaceMetrics[wl.Namespace].CPUReq, wl.Workload),
					wl.Value))
			}

			formattedDetailOut.WriteString("|===\n\n")
		}

		if len(highMemWorkloads) > 0 {
			formattedDetailOut.WriteString("=== High Memory Utilization Workloads ===\n\n")
			formattedDetailOut.WriteString("{set:cellbgcolor!}\n") // Reset any cell background color
			formattedDetailOut.WriteString("[cols=\"1,1,1,1,1\", options=\"header\"]\n|===\n")
			formattedDetailOut.WriteString("|Namespace|Workload|Memory Usage|Memory Request|Utilization\n")

			for _, wl := range highMemWorkloads {
				formattedDetailOut.WriteString(fmt.Sprintf("|%s|%s|%s|%s|%.2f%%\n",
					wl.Namespace,
					wl.Workload,
					findHumanValue(allNamespaceMetrics[wl.Namespace].MemUsage, wl.Workload),
					findHumanValue(allNamespaceMetrics[wl.Namespace].MemReq, wl.Workload),
					wl.Value))
			}

			formattedDetailOut.WriteString("|===\n\n")
		}
	}

	// Add detailed metrics for each namespace
	formattedDetailOut.WriteString("== Detailed Namespace Workload Metrics ==\n\n")

	for ns, metrics := range allNamespaceMetrics {
		formattedDetailOut.WriteString(fmt.Sprintf("=== Namespace: %s ===\n\n", ns))

		// CPU Usage
		if len(metrics.CPUUsage) > 0 {
			formattedDetailOut.WriteString("==== Top CPU Usage ====\n\n")
			formattedDetailOut.WriteString("{set:cellbgcolor!}\n") // Reset cell background color
			formattedDetailOut.WriteString("[cols=\"1,1\", options=\"header\"]\n|===\n")
			formattedDetailOut.WriteString("|Workload|CPU Usage\n")

			for i, wl := range metrics.CPUUsage {
				if i >= c.TopCount {
					break
				}
				formattedDetailOut.WriteString(fmt.Sprintf("|%s|%s\n", wl.Workload, wl.HumanValue))
			}

			formattedDetailOut.WriteString("|===\n\n")
		}

		// Memory Usage
		if len(metrics.MemUsage) > 0 {
			formattedDetailOut.WriteString("==== Top Memory Usage ====\n\n")
			formattedDetailOut.WriteString("{set:cellbgcolor!}\n") // Reset cell background color
			formattedDetailOut.WriteString("[cols=\"1,1\", options=\"header\"]\n|===\n")
			formattedDetailOut.WriteString("|Workload|Memory Usage\n")

			for i, wl := range metrics.MemUsage {
				if i >= c.TopCount {
					break
				}
				formattedDetailOut.WriteString(fmt.Sprintf("|%s|%s\n", wl.Workload, wl.HumanValue))
			}

			formattedDetailOut.WriteString("|===\n\n")
		}

		// CPU Request/Limit Comparison
		if len(metrics.CPUReq) > 0 && len(metrics.CPULim) > 0 {
			formattedDetailOut.WriteString("==== CPU Resources ====\n\n")
			formattedDetailOut.WriteString("{set:cellbgcolor!}\n") // Reset cell background color
			formattedDetailOut.WriteString("[cols=\"1,1,1,1\", options=\"header\"]\n|===\n")
			formattedDetailOut.WriteString("|Workload|CPU Usage|CPU Request|CPU Limit\n")

			// Collect combined list of workloads from usage, requests, and limits
			workloads := getUniqueWorkloads(metrics.CPUUsage, metrics.CPUReq, metrics.CPULim)

			for i, workload := range workloads {
				if i >= c.TopCount {
					break
				}

				usage := findHumanValue(metrics.CPUUsage, workload)
				req := findHumanValue(metrics.CPUReq, workload)
				lim := findHumanValue(metrics.CPULim, workload)

				formattedDetailOut.WriteString(fmt.Sprintf("|%s|%s|%s|%s\n",
					workload,
					usage,
					req,
					lim))
			}

			formattedDetailOut.WriteString("|===\n\n")
		}

		// Memory Request/Limit Comparison
		if len(metrics.MemReq) > 0 && len(metrics.MemLim) > 0 {
			formattedDetailOut.WriteString("==== Memory Resources ====\n\n")
			formattedDetailOut.WriteString("{set:cellbgcolor!}\n") // Reset cell background color
			formattedDetailOut.WriteString("[cols=\"1,1,1,1\", options=\"header\"]\n|===\n")
			formattedDetailOut.WriteString("|Workload|Memory Usage|Memory Request|Memory Limit\n")

			// Collect combined list of workloads from usage, requests, and limits
			workloads := getUniqueWorkloads(metrics.MemUsage, metrics.MemReq, metrics.MemLim)

			for i, workload := range workloads {
				if i >= c.TopCount {
					break
				}

				usage := findHumanValue(metrics.MemUsage, workload)
				req := findHumanValue(metrics.MemReq, workload)
				lim := findHumanValue(metrics.MemLim, workload)

				formattedDetailOut.WriteString(fmt.Sprintf("|%s|%s|%s|%s\n",
					workload,
					usage,
					req,
					lim))
			}

			formattedDetailOut.WriteString("|===\n\n")
		}
	}

	// Add best practices and recommendations
	formattedDetailOut.WriteString("== Resource Optimization Best Practices ==\n\n")
	formattedDetailOut.WriteString("1. **Set appropriate resource requests and limits**\n")
	formattedDetailOut.WriteString("   * Set requests based on actual application needs\n")
	formattedDetailOut.WriteString("   * Set limits to prevent resource hogging\n\n")
	formattedDetailOut.WriteString("2. **Monitor resource utilization patterns**\n")
	formattedDetailOut.WriteString("   * Use trending data to anticipate future needs\n")
	formattedDetailOut.WriteString("   * Look for cyclical patterns in resource usage\n\n")
	formattedDetailOut.WriteString("3. **Right-size workloads**\n")
	formattedDetailOut.WriteString("   * Increase requests for workloads consistently at >80% utilization\n")
	formattedDetailOut.WriteString("   * Decrease requests for workloads consistently at <30% utilization\n\n")

	// Determine result status based on problematic workloads
	var resultStatus types.Status
	var resultKey types.ResultKey
	var resultMessage string
	var recommendations []string

	if len(highCPUWorkloads) > 0 || len(highMemWorkloads) > 0 {
		var criticalWorkloads []WorkloadMetric
		var warningWorkloads []WorkloadMetric

		// Check for critical CPU utilization
		for _, wl := range highCPUWorkloads {
			if wl.Value >= c.CPUUtilizationCritical {
				criticalWorkloads = append(criticalWorkloads, wl)
			} else {
				warningWorkloads = append(warningWorkloads, wl)
			}
		}

		// Check for critical memory utilization
		for _, wl := range highMemWorkloads {
			if wl.Value >= c.MemUtilizationCritical {
				criticalWorkloads = append(criticalWorkloads, wl)
			} else {
				warningWorkloads = append(warningWorkloads, wl)
			}
		}

		if len(criticalWorkloads) > 0 {
			resultStatus = types.StatusWarning     // Changed from Critical to Warning to avoid red color
			resultKey = types.ResultKeyRecommended // Changed from Required to Recommended
			resultMessage = fmt.Sprintf("High resource utilization detected in %d user workloads", len(criticalWorkloads))
			recommendations = append(recommendations, "Increase resource requests for workloads with consistently high utilization")
			recommendations = append(recommendations, "Consider scaling out deployments horizontally for better resource distribution")
		} else {
			resultStatus = types.StatusWarning
			resultKey = types.ResultKeyRecommended
			resultMessage = fmt.Sprintf("High resource utilization detected in %d user workloads", len(warningWorkloads))
			recommendations = append(recommendations, "Monitor workloads with high utilization and consider adjusting resource requests")
		}
	} else {
		resultStatus = types.StatusOK
		resultKey = types.ResultKeyNoChange
		resultMessage = fmt.Sprintf("User workload resources are properly allocated across %d namespaces", len(allNamespaceMetrics))
	}

	// Add generic recommendations
	recommendations = append(recommendations, "Implement autoscaling for workloads with variable resource needs")
	recommendations = append(recommendations, "Consider using vertical pod autoscaler in recommendation mode to get optimal resource settings")
	recommendations = append(recommendations, "Right-size workloads by adjusting requests and limits based on actual usage patterns")

	// Create the result
	result := healthcheck.NewResult(
		c.ID(),
		resultStatus,
		resultMessage,
		resultKey,
	)

	// Add all recommendations to the result
	for _, rec := range recommendations {
		result.AddRecommendation(rec)
	}

	result.Detail = formattedDetailOut.String()
	return result, nil
}

// getUserNamespaces returns a list of user namespaces (non-OpenShift, non-kube)
func (c *UserWorkloadPerformanceCheck) getUserNamespaces(ctx context.Context) ([]string, error) {
	var userNamespaces []string

	// Get all namespaces
	nsOutput, err := utils.RunCommand("oc", "get", "namespaces", "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return nil, fmt.Errorf("failed to get namespaces: %v", err)
	}

	// Split by spaces to get individual namespaces
	namespaces := strings.Fields(nsOutput)

	// Filter out OpenShift, kube, and cert-manager namespaces
	for _, ns := range namespaces {
		if !strings.HasPrefix(ns, "openshift-") &&
			!strings.HasPrefix(ns, "kube-") &&
			!strings.HasPrefix(ns, "cert-manager") &&
			ns != "default" &&
			ns != "openshift" {
			userNamespaces = append(userNamespaces, ns)
		}
	}

	return userNamespaces, nil
}

// collectNamespaceWorkloadMetrics collects workload metrics for a namespace
func (c *UserWorkloadPerformanceCheck) collectNamespaceWorkloadMetrics(ctx context.Context, namespace, host, token string) (*NamespaceWorkloadMetrics, error) {
	baseURL := fmt.Sprintf("https://%s/api/v1/query", host)
	metrics := &NamespaceWorkloadMetrics{
		Namespace: namespace,
	}

	// Define queries with namespace
	queries := map[string]string{
		"pods":      fmt.Sprintf("count(namespace_workload_pod:kube_pod_owner:relabel{namespace=\"%s\"}) by(workload)", namespace),
		"cpu_usage": fmt.Sprintf("sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{namespace=\"%s\"} * on(namespace,pod) group_left(workload) namespace_workload_pod:kube_pod_owner:relabel{namespace=\"%s\"}) by(workload)", namespace, namespace),
		"cpu_req":   fmt.Sprintf("sum(kube_pod_container_resource_requests{job=\"kube-state-metrics\",namespace=\"%s\",resource=\"cpu\"} * on(namespace,pod) group_left(workload) namespace_workload_pod:kube_pod_owner:relabel{namespace=\"%s\"}) by(workload)", namespace, namespace),
		"cpu_lim":   fmt.Sprintf("sum(kube_pod_container_resource_limits{job=\"kube-state-metrics\",namespace=\"%s\",resource=\"cpu\"} * on(namespace,pod) group_left(workload) namespace_workload_pod:kube_pod_owner:relabel{namespace=\"%s\"}) by(workload)", namespace, namespace),
		"mem_usage": fmt.Sprintf("sum(container_memory_working_set_bytes{namespace=\"%s\",container!=\"\",image!=\"\"} * on(namespace,pod) group_left(workload) namespace_workload_pod:kube_pod_owner:relabel{namespace=\"%s\"}) by(workload)", namespace, namespace),
		"mem_req":   fmt.Sprintf("sum(kube_pod_container_resource_requests{job=\"kube-state-metrics\",namespace=\"%s\",resource=\"memory\"} * on(namespace,pod) group_left(workload) namespace_workload_pod:kube_pod_owner:relabel{namespace=\"%s\"}) by(workload)", namespace, namespace),
		"mem_lim":   fmt.Sprintf("sum(kube_pod_container_resource_limits{job=\"kube-state-metrics\",namespace=\"%s\",resource=\"memory\"} * on(namespace,pod) group_left(workload) namespace_workload_pod:kube_pod_owner:relabel{namespace=\"%s\"}) by(workload)", namespace, namespace),
	}

	// Set up HTTP client with timeout
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}

	// Execute queries
	for name, query := range queries {
		// Set up URL and parameters
		u, err := url.Parse(baseURL)
		if err != nil {
			continue
		}

		params := url.Values{}
		params.Add("query", query)
		u.RawQuery = params.Encode()

		// Create request
		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			continue
		}
		req.Header.Add("Authorization", "Bearer "+token)

		// Execute request
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		// Read response
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		// Parse the JSON result
		var result struct {
			Data struct {
				Result []struct {
					Metric map[string]string `json:"metric"`
					Value  []interface{}     `json:"value"`
				} `json:"result"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			continue
		}

		// Extract values and store in appropriate metrics slice
		workloadMetrics := make([]WorkloadMetric, 0, len(result.Data.Result))
		for _, r := range result.Data.Result {
			workloadName := r.Metric["workload"]
			if len(r.Value) < 2 {
				continue
			}

			valueStr, ok := r.Value[1].(string)
			if !ok {
				continue
			}

			value, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				continue
			}

			humanValue := ""
			switch name {
			case "pods":
				humanValue = fmt.Sprintf("%d pods", int(value))
			case "cpu_usage", "cpu_req", "cpu_lim":
				humanValue = fmt.Sprintf("%.0f m", value*1000)
			case "mem_usage", "mem_req", "mem_lim":
				humanValue = fmt.Sprintf("%.2f MiB", value/1024/1024)
			}

			workloadMetrics = append(workloadMetrics, WorkloadMetric{
				Namespace:  namespace,
				Workload:   workloadName,
				Value:      value,
				HumanValue: humanValue,
			})
		}

		// Sort metrics by value in descending order
		sort.Slice(workloadMetrics, func(i, j int) bool {
			return workloadMetrics[i].Value > workloadMetrics[j].Value
		})

		// Store in appropriate field
		switch name {
		case "pods":
			metrics.Pods = workloadMetrics
		case "cpu_usage":
			metrics.CPUUsage = workloadMetrics
		case "cpu_req":
			metrics.CPUReq = workloadMetrics
		case "cpu_lim":
			metrics.CPULim = workloadMetrics
		case "mem_usage":
			metrics.MemUsage = workloadMetrics
		case "mem_req":
			metrics.MemReq = workloadMetrics
		case "mem_lim":
			metrics.MemLim = workloadMetrics
		}
	}

	// Calculate derived metrics like utilization percentages
	metrics.CPUUtilization = c.calculateUtilization(metrics.CPUUsage, metrics.CPUReq)
	metrics.MemUtilization = c.calculateUtilization(metrics.MemUsage, metrics.MemReq)

	return metrics, nil
}

// calculateUtilization calculates the utilization percentage of usage vs. request
func (c *UserWorkloadPerformanceCheck) calculateUtilization(usage, requests []WorkloadMetric) []WorkloadMetric {
	// Create maps for quick lookups
	usageMap := make(map[string]float64)
	for _, u := range usage {
		usageMap[u.Workload] = u.Value
	}

	reqMap := make(map[string]float64)
	for _, r := range requests {
		reqMap[r.Workload] = r.Value
	}

	// Calculate utilization for workloads with both usage and request
	var utilizations []WorkloadMetric
	for workload, usage := range usageMap {
		if req, ok := reqMap[workload]; ok && req > 0 {
			utilizations = append(utilizations, WorkloadMetric{
				Namespace:  "", // Will be filled in later when needed
				Workload:   workload,
				Value:      (usage / req) * 100,
				HumanValue: fmt.Sprintf("%.2f%%", (usage/req)*100),
			})
		}
	}

	// Sort by utilization in descending order
	sort.Slice(utilizations, func(i, j int) bool {
		return utilizations[i].Value > utilizations[j].Value
	})

	return utilizations
}

// identifyProblematicWorkloads finds workloads with high CPU or memory utilization
func (c *UserWorkloadPerformanceCheck) identifyProblematicWorkloads(
	allNamespaceMetrics map[string]*NamespaceWorkloadMetrics) ([]WorkloadMetric, []WorkloadMetric) {

	var highCPUWorkloads []WorkloadMetric
	var highMemWorkloads []WorkloadMetric

	for ns, metrics := range allNamespaceMetrics {
		// Check CPU utilization
		for _, wl := range metrics.CPUUtilization {
			if wl.Value >= c.CPUUtilizationWarning {
				// Create a copy to avoid modifying the original
				problemWL := WorkloadMetric{
					Namespace:  ns,
					Workload:   wl.Workload,
					Value:      wl.Value,
					HumanValue: wl.HumanValue,
				}
				highCPUWorkloads = append(highCPUWorkloads, problemWL)
			}
		}

		// Check memory utilization
		for _, wl := range metrics.MemUtilization {
			if wl.Value >= c.MemUtilizationWarning {
				// Create a copy to avoid modifying the original
				problemWL := WorkloadMetric{
					Namespace:  ns,
					Workload:   wl.Workload,
					Value:      wl.Value,
					HumanValue: wl.HumanValue,
				}
				highMemWorkloads = append(highMemWorkloads, problemWL)
			}
		}
	}

	// Sort by utilization percentage (highest first)
	sort.Slice(highCPUWorkloads, func(i, j int) bool {
		return highCPUWorkloads[i].Value > highCPUWorkloads[j].Value
	})

	sort.Slice(highMemWorkloads, func(i, j int) bool {
		return highMemWorkloads[i].Value > highMemWorkloads[j].Value
	})

	return highCPUWorkloads, highMemWorkloads
}

// findHumanValue returns the human-readable value for a workload from a metrics slice
func findHumanValue(metrics []WorkloadMetric, workload string) string {
	for _, m := range metrics {
		if m.Workload == workload {
			return m.HumanValue
		}
	}
	return "N/A"
}

// getUniqueWorkloads returns a sorted, unique list of workload names from multiple metrics slices
func getUniqueWorkloads(metricSlices ...[]WorkloadMetric) []string {
	workloadMap := make(map[string]bool)

	// Add workloads from all slices to the map
	for _, metrics := range metricSlices {
		for _, m := range metrics {
			workloadMap[m.Workload] = true
		}
	}

	// Convert map keys to slice
	var workloads []string
	for wl := range workloadMap {
		workloads = append(workloads, wl)
	}

	// Sort workloads alphabetically
	sort.Strings(workloads)

	return workloads
}
