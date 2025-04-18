/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-18

This file implements health checks for cluster performance metrics. It:

- Monitors CPU and memory utilization across the cluster
- Examines CPU and memory requests/limits commitment
- Provides detailed per-node resource utilization metrics using oc adm top
- Lists top CPU and memory consuming pods in the cluster
- Provides recommendations for improving cluster performance and resource allocation
- Uses a combination of Prometheus and CLI tools for maximum reliability

This check ensures the cluster is operating efficiently and has sufficient resources for workloads.
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

// NodeMetrics holds resource metrics for a single node
type NodeMetrics struct {
	Name              string
	CPUCapacity       string
	CPUUsage          string
	CPUUtilization    float64
	MemoryCapacity    string
	MemoryUsage       string
	MemoryUtilization float64
}

// PodMetrics holds resource metrics for a single pod
type PodMetrics struct {
	Name      string
	Namespace string
	CPU       string
	CPUValue  float64
	Memory    string
	MemValue  float64
}

// PerformanceMetrics holds various cluster performance metrics
type PerformanceMetrics struct {
	// CPU metrics
	CPUUtilization        float64
	CPURequestsCommitment float64
	CPULimitsCommitment   float64

	// Memory metrics
	MemoryUtilization        float64
	MemoryRequestsCommitment float64
	MemoryLimitsCommitment   float64

	// Node-specific metrics
	NodeMetrics []NodeMetrics

	// Top resource consumers
	TopCPUPods    []PodMetrics
	TopMemoryPods []PodMetrics

	// Raw query results for debugging
	RawResults map[string]string
}

// PrometheusQueryResult represents the structure of a Prometheus query response
type PrometheusQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// ClusterPerformanceCheck checks the performance metrics of the cluster
type ClusterPerformanceCheck struct {
	healthcheck.BaseCheck
	// Thresholds for warnings and critical alerts
	CPUUtilizationWarning     float64
	CPUUtilizationCritical    float64
	MemoryUtilizationWarning  float64
	MemoryUtilizationCritical float64
}

// NewClusterPerformanceCheck creates a new cluster performance check
func NewClusterPerformanceCheck() *ClusterPerformanceCheck {
	return &ClusterPerformanceCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"cluster-performance",
			"Cluster Performance",
			"Checks cluster performance metrics including CPU and memory utilization",
			types.CategoryPerformance,
		),
		CPUUtilizationWarning:     80.0,
		CPUUtilizationCritical:    90.0,
		MemoryUtilizationWarning:  80.0,
		MemoryUtilizationCritical: 90.0,
	}
}

// Run executes the health check
func (c *ClusterPerformanceCheck) Run() (healthcheck.Result, error) {
	// Create a timeout context to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Collect all performance metrics
	metrics, err := c.collectPerformanceMetrics(ctx)
	if err != nil {
		// Continue with any metrics we could collect
		fmt.Printf("Warning: %v\n", err)
	}

	// Build detailed output with proper AsciiDoc formatting
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Cluster Performance Analysis ===\n\n")

	// Add overall metrics
	formattedDetailOut.WriteString("== Resource Utilization Overview ==\n\n")
	formattedDetailOut.WriteString(fmt.Sprintf("CPU Utilization: %.2f%%\n", metrics.CPUUtilization))
	formattedDetailOut.WriteString(fmt.Sprintf("CPU Requests Commitment: %.2f%%\n", metrics.CPURequestsCommitment))
	formattedDetailOut.WriteString(fmt.Sprintf("CPU Limits Commitment: %.2f%%\n", metrics.CPULimitsCommitment))
	formattedDetailOut.WriteString(fmt.Sprintf("Memory Utilization: %.2f%%\n", metrics.MemoryUtilization))
	formattedDetailOut.WriteString(fmt.Sprintf("Memory Requests Commitment: %.2f%%\n", metrics.MemoryRequestsCommitment))
	formattedDetailOut.WriteString(fmt.Sprintf("Memory Limits Commitment: %.2f%%\n\n", metrics.MemoryLimitsCommitment))

	// Add node-specific metrics
	formattedDetailOut.WriteString("== Node Resource Utilization ==\n\n")

	if len(metrics.NodeMetrics) > 0 {
		// Set cell background to none for this table
		formattedDetailOut.WriteString("{set:cellbgcolor!}\n")
		formattedDetailOut.WriteString("[cols=\"1,1,1,1,1,1,1\", options=\"header\"]\n|===\n")
		formattedDetailOut.WriteString("|Node Name|CPU Capacity|CPU Usage|CPU %|Memory Capacity|Memory Usage|Memory %\n")

		for _, node := range metrics.NodeMetrics {
			formattedDetailOut.WriteString(fmt.Sprintf("|%s|%s|%s|%.2f%%|%s|%s|%.2f%%\n",
				node.Name,
				node.CPUCapacity,
				node.CPUUsage,
				node.CPUUtilization,
				node.MemoryCapacity,
				node.MemoryUsage,
				node.MemoryUtilization))
		}

		formattedDetailOut.WriteString("|===\n\n")

		// Identify high utilization nodes
		var highCpuNodes []string
		var highMemNodes []string

		for _, node := range metrics.NodeMetrics {
			if node.CPUUtilization >= c.CPUUtilizationWarning {
				highCpuNodes = append(highCpuNodes, fmt.Sprintf("%s (%.2f%%)", node.Name, node.CPUUtilization))
			}
			if node.MemoryUtilization >= c.MemoryUtilizationWarning {
				highMemNodes = append(highMemNodes, fmt.Sprintf("%s (%.2f%%)", node.Name, node.MemoryUtilization))
			}
		}

		if len(highCpuNodes) > 0 {
			formattedDetailOut.WriteString("=== Nodes with High CPU Utilization ===\n\n")
			for _, node := range highCpuNodes {
				formattedDetailOut.WriteString(fmt.Sprintf("* %s\n", node))
			}
			formattedDetailOut.WriteString("\n")
		}

		if len(highMemNodes) > 0 {
			formattedDetailOut.WriteString("=== Nodes with High Memory Utilization ===\n\n")
			for _, node := range highMemNodes {
				formattedDetailOut.WriteString(fmt.Sprintf("* %s\n", node))
			}
			formattedDetailOut.WriteString("\n")
		}
	} else {
		formattedDetailOut.WriteString("No node-specific metrics available.\n\n")
	}

	// Add top CPU and memory consumers
	if len(metrics.TopCPUPods) > 0 {
		formattedDetailOut.WriteString("== Top CPU Consumers (Top 10) ==\n\n")
		formattedDetailOut.WriteString("[cols=\"1,1,1\", options=\"header\"]\n|===\n")
		formattedDetailOut.WriteString("|Pod Name|Namespace|CPU Usage\n\n")

		for _, pod := range metrics.TopCPUPods {
			formattedDetailOut.WriteString(fmt.Sprintf("|%s|%s|%s\n",
				pod.Name,
				pod.Namespace,
				pod.CPU))
		}

		formattedDetailOut.WriteString("|===\n\n")
	}

	if len(metrics.TopMemoryPods) > 0 {
		formattedDetailOut.WriteString("== Top Memory Consumers (Top 10) ==\n\n")
		formattedDetailOut.WriteString("[cols=\"1,1,1\", options=\"header\"]\n|===\n")
		formattedDetailOut.WriteString("|Pod Name|Namespace|Memory Usage\n\n")

		for _, pod := range metrics.TopMemoryPods {
			formattedDetailOut.WriteString(fmt.Sprintf("|%s|%s|%s\n",
				pod.Name,
				pod.Namespace,
				pod.Memory))
		}

		formattedDetailOut.WriteString("|===\n\n")
	}

	// If we have raw results, include them for debugging
	if len(metrics.RawResults) > 0 {
		formattedDetailOut.WriteString("== Raw Query Results ==\n\n")
		formattedDetailOut.WriteString("[source,json]\n----\n")
		for query, result := range metrics.RawResults {
			formattedDetailOut.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", query, result))
		}
		formattedDetailOut.WriteString("----\n\n")
	}

	// Determine result status based on thresholds
	var resultStatus types.Status
	var resultKey types.ResultKey
	var resultMessage string
	var recommendations []string

	// Check CPU utilization against thresholds
	if metrics.CPUUtilization >= c.CPUUtilizationCritical {
		resultStatus = types.StatusCritical
		resultKey = types.ResultKeyRequired
		resultMessage = fmt.Sprintf("Critical CPU utilization (%.2f%%) exceeds threshold (%.2f%%)",
			metrics.CPUUtilization, c.CPUUtilizationCritical)
		recommendations = append(recommendations, "Add capacity to the cluster by scaling up or adding more nodes")
		recommendations = append(recommendations, "Investigate high CPU workloads and consider setting appropriate resource limits")
	} else if metrics.CPUUtilization >= c.CPUUtilizationWarning {
		resultStatus = types.StatusWarning
		resultKey = types.ResultKeyRecommended
		resultMessage = fmt.Sprintf("High CPU utilization (%.2f%%) exceeds warning threshold (%.2f%%)",
			metrics.CPUUtilization, c.CPUUtilizationWarning)
		recommendations = append(recommendations, "Plan for additional capacity in the cluster")
		recommendations = append(recommendations, "Review workload resource requests and limits")
	} else if metrics.MemoryUtilization >= c.MemoryUtilizationCritical {
		resultStatus = types.StatusCritical
		resultKey = types.ResultKeyRequired
		resultMessage = fmt.Sprintf("Critical memory utilization (%.2f%%) exceeds threshold (%.2f%%)",
			metrics.MemoryUtilization, c.MemoryUtilizationCritical)
		recommendations = append(recommendations, "Add capacity to the cluster by scaling up memory or adding more nodes")
		recommendations = append(recommendations, "Investigate high memory workloads and consider optimizing applications")
	} else if metrics.MemoryUtilization >= c.MemoryUtilizationWarning {
		resultStatus = types.StatusWarning
		resultKey = types.ResultKeyRecommended
		resultMessage = fmt.Sprintf("High memory utilization (%.2f%%) exceeds warning threshold (%.2f%%)",
			metrics.MemoryUtilization, c.MemoryUtilizationWarning)
		recommendations = append(recommendations, "Plan for additional memory capacity in the cluster")
		recommendations = append(recommendations, "Review and optimize memory-intensive workloads")
	} else {
		resultStatus = types.StatusOK
		resultKey = types.ResultKeyNoChange
		resultMessage = fmt.Sprintf("Cluster performance metrics are within acceptable ranges (CPU: %.2f%%, Memory: %.2f%%)",
			metrics.CPUUtilization, metrics.MemoryUtilization)
	}

	// Create the result
	result := healthcheck.NewResult(
		c.ID(),
		resultStatus,
		resultMessage,
		resultKey,
	)

	// Add best practices recommendations regardless of status
	recommendations = append(recommendations, "Ensure resource requests and limits are set appropriately for all workloads")
	recommendations = append(recommendations, "Consider using horizontal pod autoscaling for applications with variable workloads")
	recommendations = append(recommendations, "Separate infrastructure workloads from application workloads using node selectors and taints")

	// Add all recommendations to the result
	for _, rec := range recommendations {
		result.AddRecommendation(rec)
	}

	result.Detail = formattedDetailOut.String()
	return result, nil
}

// collectPerformanceMetrics gathers all performance metrics from the cluster
func (c *ClusterPerformanceCheck) collectPerformanceMetrics(ctx context.Context) (*PerformanceMetrics, error) {
	metrics := &PerformanceMetrics{
		RawResults: make(map[string]string),
	}

	// Define queries
	queries := map[string]string{
		"cpu_utilization":            "cluster:node_cpu:ratio_rate5m{}",
		"cpu_requests_commitment":    "sum(namespace_cpu:kube_pod_container_resource_requests:sum{}) / sum(kube_node_status_allocatable{job=\"kube-state-metrics\",resource=\"cpu\"})",
		"cpu_limits_commitment":      "sum(namespace_cpu:kube_pod_container_resource_limits:sum{})  / sum(kube_node_status_allocatable{job=\"kube-state-metrics\",resource=\"cpu\"})",
		"memory_utilization":         "1 - sum(:node_memory_MemAvailable_bytes:sum{}) / sum(node_memory_MemTotal_bytes{job=\"node-exporter\"})",
		"memory_requests_commitment": "sum(namespace_memory:kube_pod_container_resource_requests:sum{}) / sum(kube_node_status_allocatable{job=\"kube-state-metrics\",resource=\"memory\"})",
		"memory_limits_commitment":   "sum(namespace_memory:kube_pod_container_resource_limits:sum{})  / sum(kube_node_status_allocatable{job=\"kube-state-metrics\",resource=\"memory\"})",
	}

	// Get token for authentication
	token, err := utils.RunCommand("oc", "whoami", "-t")
	if err != nil {
		// Continue silently if can't get token
		token = ""
	}
	token = strings.TrimSpace(token)

	// Get thanos querier host
	host, err := utils.RunCommand("oc", "-n", "openshift-monitoring", "get", "route", "thanos-querier", "-o", "jsonpath={.status.ingress[0].host}")
	if err != nil {
		// Continue silently if can't get host
		host = ""
	}
	host = strings.TrimSpace(host)

	if token != "" && host != "" {
		baseURL := fmt.Sprintf("https://%s/api/v1/query", host)

		// Variables to track values
		var cpuUtil, cpuReq, cpuLim, memUtil, memReq, memLim float64

		// Execute queries
		for name, query := range queries {
			// Set up HTTP client with timeout
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{Transport: tr, Timeout: 10 * time.Second}

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

			// Store raw result for debugging
			metrics.RawResults[name] = string(body)

			// Parse the JSON result
			var result PrometheusQueryResult
			if err := json.Unmarshal(body, &result); err != nil {
				continue
			}

			// Extract the value
			if result.Status != "success" || len(result.Data.Result) == 0 {
				continue
			}

			if len(result.Data.Result[0].Value) < 2 {
				continue
			}

			valueStr, ok := result.Data.Result[0].Value[1].(string)
			if !ok {
				continue
			}

			value, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				continue
			}

			// Assign value to appropriate metric
			switch name {
			case "cpu_utilization":
				cpuUtil = value * 100 // Convert to percentage
			case "cpu_requests_commitment":
				cpuReq = value * 100 // Convert to percentage
			case "cpu_limits_commitment":
				cpuLim = value * 100 // Convert to percentage
			case "memory_utilization":
				memUtil = value * 100 // Convert to percentage
			case "memory_requests_commitment":
				memReq = value * 100 // Convert to percentage
			case "memory_limits_commitment":
				memLim = value * 100 // Convert to percentage
			}
		}

		// Set metrics values
		metrics.CPUUtilization = cpuUtil
		metrics.CPURequestsCommitment = cpuReq
		metrics.CPULimitsCommitment = cpuLim
		metrics.MemoryUtilization = memUtil
		metrics.MemoryRequestsCommitment = memReq
		metrics.MemoryLimitsCommitment = memLim
	}

	// Get node utilization using 'oc adm top nodes'
	nodeMetrics, err := c.getNodeUtilization(ctx)
	if err != nil {
		// Silently continue without node metrics
		nodeMetrics = []NodeMetrics{}
	}
	metrics.NodeMetrics = nodeMetrics

	// Get top pod CPU and memory consumers
	topCPUPods, err := c.getTopPods(ctx, "cpu")
	if err != nil {
		// Silently continue without top CPU pods
		topCPUPods = []PodMetrics{}
	}
	metrics.TopCPUPods = topCPUPods

	topMemoryPods, err := c.getTopPods(ctx, "memory")
	if err != nil {
		// Silently continue without top memory pods
		topMemoryPods = []PodMetrics{}
	}
	metrics.TopMemoryPods = topMemoryPods

	return metrics, nil
}

// getNodeUtilization retrieves node utilization using 'oc adm top nodes'
func (c *ClusterPerformanceCheck) getNodeUtilization(ctx context.Context) ([]NodeMetrics, error) {
	var nodes []NodeMetrics

	// Get node utilization using 'oc adm top nodes'
	out, err := utils.RunCommand("oc", "adm", "top", "nodes")
	if err != nil {
		return nodes, fmt.Errorf("failed to get node utilization information: %v", err)
	}

	// Parse the output
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		return nodes, fmt.Errorf("unexpected output format from 'oc adm top nodes'")
	}

	// Skip header line
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// Fields from 'oc adm top nodes' are:
		// NAME CPU(cores) CPU% MEMORY(bytes) MEMORY%

		cpuUtilization, _ := strconv.ParseFloat(strings.TrimSuffix(fields[2], "%"), 64)
		memUtilization, _ := strconv.ParseFloat(strings.TrimSuffix(fields[4], "%"), 64)

		node := NodeMetrics{
			Name:              fields[0],
			CPUCapacity:       "", // Will be populated below
			CPUUsage:          fields[1],
			CPUUtilization:    cpuUtilization,
			MemoryCapacity:    "", // Will be populated below
			MemoryUsage:       fields[3],
			MemoryUtilization: memUtilization,
		}

		nodes = append(nodes, node)
	}

	// Get capacity information to supplement the metrics
	capacityOut, err := utils.RunCommand("oc", "get", "nodes", "-o", "custom-columns=NAME:.metadata.name,CPU:.status.capacity.cpu,MEMORY:.status.capacity.memory")
	if err == nil {
		capacityLines := strings.Split(capacityOut, "\n")
		if len(capacityLines) >= 2 {
			// Create a map for quick lookups
			capacityMap := make(map[string][]string)

			// Skip header line
			for i := 1; i < len(capacityLines); i++ {
				line := strings.TrimSpace(capacityLines[i])
				if line == "" {
					continue
				}

				fields := strings.Fields(line)
				if len(fields) < 3 {
					continue
				}

				capacityMap[fields[0]] = []string{fields[1], fields[2]}
			}

			// Update node metrics with capacity information
			for i := range nodes {
				if capacity, ok := capacityMap[nodes[i].Name]; ok && len(capacity) >= 2 {
					nodes[i].CPUCapacity = capacity[0]
					nodes[i].MemoryCapacity = capacity[1]
				}
			}
		}
	}

	// Sort nodes by CPU utilization (highest first)
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].CPUUtilization > nodes[j].CPUUtilization
	})

	return nodes, nil
}

// getTopPods retrieves top pod consumers for CPU or memory
func (c *ClusterPerformanceCheck) getTopPods(ctx context.Context, resource string) ([]PodMetrics, error) {
	var pods []PodMetrics

	// Get top pods using 'oc adm top pods'
	out, err := utils.RunCommand("oc", "adm", "top", "pods", "--all-namespaces", fmt.Sprintf("--sort-by=%s", resource))
	if err != nil {
		return pods, fmt.Errorf("failed to get top pods information: %v", err)
	}

	// Parse the output
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		return pods, fmt.Errorf("unexpected output format from 'oc adm top pods'")
	}

	// Skip header line and limit to top 10
	count := 0
	for i := 1; i < len(lines) && count < 10; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// Fields from 'oc adm top pods' are:
		// NAMESPACE NAME CPU(cores) MEMORY(bytes)

		var cpuValue, memValue float64

		// Try to parse CPU value for sorting
		cpuStr := fields[2]
		if strings.HasSuffix(cpuStr, "m") {
			// Convert millicores to cores
			mcores, _ := strconv.ParseFloat(strings.TrimSuffix(cpuStr, "m"), 64)
			cpuValue = mcores / 1000
		} else {
			cpuValue, _ = strconv.ParseFloat(cpuStr, 64)
		}

		// Try to parse memory value for sorting
		memStr := fields[3]
		if strings.HasSuffix(memStr, "Mi") {
			memValue, _ = strconv.ParseFloat(strings.TrimSuffix(memStr, "Mi"), 64)
		} else if strings.HasSuffix(memStr, "Gi") {
			memValue, _ = strconv.ParseFloat(strings.TrimSuffix(memStr, "Gi"), 64)
			memValue *= 1024 // Convert to Mi
		} else if strings.HasSuffix(memStr, "Ki") {
			memValue, _ = strconv.ParseFloat(strings.TrimSuffix(memStr, "Ki"), 64)
			memValue /= 1024 // Convert to Mi
		} else {
			memValue, _ = strconv.ParseFloat(memStr, 64)
		}

		pod := PodMetrics{
			Namespace: fields[0],
			Name:      fields[1],
			CPU:       fields[2],
			CPUValue:  cpuValue,
			Memory:    fields[3],
			MemValue:  memValue,
		}

		pods = append(pods, pod)
		count++
	}

	return pods, nil
}
