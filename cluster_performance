/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-17

This file implements health checks for cluster performance metrics. It:

- Monitors CPU and memory utilization across the cluster
- Examines CPU and memory requests/limits commitment
- Tracks network bandwidth and throughput metrics
- Analyzes storage I/O performance
- Provides recommendations for improving cluster performance and resource allocation
- Helps identify potential bottlenecks before they affect application performance

This check ensures the cluster is operating efficiently and has sufficient resources for workloads.
*/

package performance

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"strconv"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

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

	// Namespace resource usage
	NamespaceMetrics map[string]*NamespaceMetrics

	// Node metrics
	NodeMetrics map[string]*NodeMetrics

	// Network metrics
	NetworkReceiveBandwidth  float64 // in MBps
	NetworkTransmitBandwidth float64 // in MBps

	// Storage metrics
	StorageIOPS       float64
	StorageThroughput float64 // in MBps
}

// NamespaceMetrics holds resource metrics for a namespace
type NamespaceMetrics struct {
	Name string

	// CPU metrics
	CPUUsage    float64 // in cores
	CPURequests float64 // in cores
	CPULimits   float64 // in cores

	// Memory metrics
	MemoryUsage    float64 // in bytes
	MemoryRequests float64 // in bytes
	MemoryLimits   float64 // in bytes

	// Network metrics
	NetworkReceiveBandwidth  float64 // in Bps
	NetworkTransmitBandwidth float64 // in Bps
}

// NodeMetrics holds resource metrics for a node
type NodeMetrics struct {
	Name string

	// CPU metrics
	CPUCapacity    float64 // in cores
	CPUAllocatable float64 // in cores
	CPUUsage       float64 // in cores

	// Memory metrics
	MemoryCapacity    float64 // in bytes
	MemoryAllocatable float64 // in bytes
	MemoryUsage       float64 // in bytes
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
			"Checks cluster performance metrics including CPU, memory, network, and storage utilization",
			types.CategoryClusterConfig,
		),
		CPUUtilizationWarning:     80.0,
		CPUUtilizationCritical:    90.0,
		MemoryUtilizationWarning:  80.0,
		MemoryUtilizationCritical: 90.0,
	}
}

// Run executes the health check
func (c *ClusterPerformanceCheck) Run() (healthcheck.Result, error) {
	// Get Kubernetes client
	clientset, err := utils.GetClientSet()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get Kubernetes client",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting Kubernetes client: %v", err)
	}

	// Get dynamic client for metrics
	config, err := utils.GetClusterConfig()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get cluster configuration",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting cluster configuration: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to create dynamic client",
			types.ResultKeyRequired,
		), fmt.Errorf("error creating dynamic client: %v", err)
	}

	// Collect all performance metrics
	metrics, err := c.collectPerformanceMetrics(clientset, dynamicClient)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Failed to collect some performance metrics: %v", err),
			types.ResultKeyAdvisory,
		), nil
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

	// Add detailed information from namespace monitoring
	topNsCPU, topNsMem := c.getTopNamespaces(metrics, 10)
	formattedDetailOut.WriteString("== Top Namespaces by Resource Usage ==\n\n")

	// Top CPU consumers
	formattedDetailOut.WriteString("=== Top CPU Consumers ===\n\n")
	formattedDetailOut.WriteString("[cols=\"1,1,1,1,1,1\", options=\"header\"]\n|===\n")
	formattedDetailOut.WriteString("|Namespace|CPU Usage (cores)|CPU Requests (cores)|Usage/Requests|CPU Limits (cores)|Usage/Limits\n\n")

	for _, nsName := range topNsCPU {
		ns := metrics.NamespaceMetrics[nsName]
		reqRatio := 0.0
		if ns.CPURequests > 0 {
			reqRatio = (ns.CPUUsage / ns.CPURequests) * 100
		}

		limRatio := 0.0
		if ns.CPULimits > 0 {
			limRatio = (ns.CPUUsage / ns.CPULimits) * 100
		}

		formattedDetailOut.WriteString(fmt.Sprintf("|%s|%.3f|%.3f|%.1f%%|%.3f|%.1f%%\n",
			ns.Name, ns.CPUUsage, ns.CPURequests, reqRatio, ns.CPULimits, limRatio))
	}
	formattedDetailOut.WriteString("|===\n\n")

	// Top Memory consumers
	formattedDetailOut.WriteString("=== Top Memory Consumers ===\n\n")
	formattedDetailOut.WriteString("[cols=\"1,1,1,1,1,1\", options=\"header\"]\n|===\n")
	formattedDetailOut.WriteString("|Namespace|Memory Usage|Memory Requests|Usage/Requests|Memory Limits|Usage/Limits\n\n")

	for _, nsName := range topNsMem {
		ns := metrics.NamespaceMetrics[nsName]
		reqRatio := 0.0
		if ns.MemoryRequests > 0 {
			reqRatio = (ns.MemoryUsage / ns.MemoryRequests) * 100
		}

		limRatio := 0.0
		if ns.MemoryLimits > 0 {
			limRatio = (ns.MemoryUsage / ns.MemoryLimits) * 100
		}

		formattedDetailOut.WriteString(fmt.Sprintf("|%s|%s|%s|%.1f%%|%s|%.1f%%\n",
			ns.Name,
			formatBytes(ns.MemoryUsage),
			formatBytes(ns.MemoryRequests),
			reqRatio,
			formatBytes(ns.MemoryLimits),
			limRatio))
	}
	formattedDetailOut.WriteString("|===\n\n")

	// Add node utilization information
	formattedDetailOut.WriteString("== Node Resource Utilization ==\n\n")
	formattedDetailOut.WriteString("[cols=\"1,1,1,1,1,1,1\", options=\"header\"]\n|===\n")
	formattedDetailOut.WriteString("|Node|CPU Usage|CPU Capacity|CPU %|Memory Usage|Memory Capacity|Memory %\n\n")

	for nodeName, nodeMetric := range metrics.NodeMetrics {
		cpuPercent := 0.0
		if nodeMetric.CPUCapacity > 0 {
			cpuPercent = (nodeMetric.CPUUsage / nodeMetric.CPUCapacity) * 100
		}

		memPercent := 0.0
		if nodeMetric.MemoryCapacity > 0 {
			memPercent = (nodeMetric.MemoryUsage / nodeMetric.MemoryCapacity) * 100
		}

		formattedDetailOut.WriteString(fmt.Sprintf("|%s|%.2f cores|%.2f cores|%.1f%%|%s|%s|%.1f%%\n",
			nodeName,
			nodeMetric.CPUUsage,
			nodeMetric.CPUCapacity,
			cpuPercent,
			formatBytes(nodeMetric.MemoryUsage),
			formatBytes(nodeMetric.MemoryCapacity),
			memPercent))
	}
	formattedDetailOut.WriteString("|===\n\n")

	// Add network utilization if available
	if metrics.NetworkReceiveBandwidth > 0 || metrics.NetworkTransmitBandwidth > 0 {
		formattedDetailOut.WriteString("== Network Utilization ==\n\n")
		formattedDetailOut.WriteString(fmt.Sprintf("Current Receive Bandwidth: %.2f MBps\n", metrics.NetworkReceiveBandwidth))
		formattedDetailOut.WriteString(fmt.Sprintf("Current Transmit Bandwidth: %.2f MBps\n\n", metrics.NetworkTransmitBandwidth))
	}

	// Add storage metrics if available
	if metrics.StorageIOPS > 0 || metrics.StorageThroughput > 0 {
		formattedDetailOut.WriteString("== Storage Performance ==\n\n")
		formattedDetailOut.WriteString(fmt.Sprintf("Current IOPS: %.2f\n", metrics.StorageIOPS))
		formattedDetailOut.WriteString(fmt.Sprintf("Current Throughput: %.2f MBps\n\n", metrics.StorageThroughput))
	}

	// Add the raw data from prometheus queries in AsciiDoc format
	prometheusQueryOut, _ := utils.RunCommand("oc", "exec", "-n", "openshift-monitoring", "prometheus-k8s-0", "-c", "prometheus", "--", "curl", "-s", "http://localhost:9090/api/v1/query?query=cluster:memory_usage:ratio")
	if strings.TrimSpace(prometheusQueryOut) != "" {
		formattedDetailOut.WriteString("== Raw Prometheus Data ==\n\n")
		formattedDetailOut.WriteString("Memory Usage Query Result:\n[source, json]\n----\n")
		formattedDetailOut.WriteString(prometheusQueryOut)
		formattedDetailOut.WriteString("\n----\n\n")
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
func (c *ClusterPerformanceCheck) collectPerformanceMetrics(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface) (*PerformanceMetrics, error) {
	metrics := &PerformanceMetrics{
		NamespaceMetrics: make(map[string]*NamespaceMetrics),
		NodeMetrics:      make(map[string]*NodeMetrics),
	}

	// First, try to get metrics directly from the Prometheus API
	err := c.getMetricsFromPrometheus(metrics)
	if err != nil {
		// Fall back to metrics server API if Prometheus is not available
		err = c.getMetricsFromMetricsServer(dynamicClient, metrics)
		if err != nil {
			// Use oc commands as last resort
			err = c.getMetricsFromOCCommands(metrics)
			if err != nil {
				return metrics, fmt.Errorf("failed to collect metrics: %v", err)
			}
		}
	}

	// Get node information to calculate capacities
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return metrics, fmt.Errorf("failed to get nodes: %v", err)
	}

	// Process node information for capacity calculations
	c.processNodeInformation(nodes.Items, metrics)

	// Calculate commitment ratios
	c.calculateCommitmentRatios(metrics)

	return metrics, nil
}

// getMetricsFromPrometheus tries to get metrics directly from Prometheus
func (c *ClusterPerformanceCheck) getMetricsFromPrometheus(metrics *PerformanceMetrics) error {
	// Queries for Prometheus
	cpuUtilQuery := "cluster:cpu_usage:ratio * 100"
	memUtilQuery := "cluster:memory_usage:ratio * 100"

	// Try to execute queries
	cpuOutput, err := utils.RunCommand("oc", "exec", "-n", "openshift-monitoring", "prometheus-k8s-0", "-c", "prometheus", "--",
		"curl", "-s", fmt.Sprintf("http://localhost:9090/api/v1/query?query=%s", cpuUtilQuery))

	if err != nil {
		return fmt.Errorf("failed to query Prometheus for CPU utilization: %v", err)
	}

	// Extract CPU utilization value - this is simplified; in a real implementation,
	// you would parse the JSON response properly
	if strings.Contains(cpuOutput, "\"value\"") {
		parts := strings.Split(cpuOutput, "\"value\":[")
		if len(parts) > 1 {
			valueStr := strings.Split(strings.Split(parts[1], "]")[0], ",")[1]
			valueStr = strings.Trim(valueStr, "\" ")
			cpuUtil, err := strconv.ParseFloat(valueStr, 64)
			if err == nil {
				metrics.CPUUtilization = cpuUtil
			}
		}
	}

	// Now for memory
	memOutput, err := utils.RunCommand("oc", "exec", "-n", "openshift-monitoring", "prometheus-k8s-0", "-c", "prometheus", "--",
		"curl", "-s", fmt.Sprintf("http://localhost:9090/api/v1/query?query=%s", memUtilQuery))

	if err != nil {
		return fmt.Errorf("failed to query Prometheus for memory utilization: %v", err)
	}

	// Extract memory utilization value
	if strings.Contains(memOutput, "\"value\"") {
		parts := strings.Split(memOutput, "\"value\":[")
		if len(parts) > 1 {
			valueStr := strings.Split(strings.Split(parts[1], "]")[0], ",")[1]
			valueStr = strings.Trim(valueStr, "\" ")
			memUtil, err := strconv.ParseFloat(valueStr, 64)
			if err == nil {
				metrics.MemoryUtilization = memUtil
			}
		}
	}

	// Get namespace metrics
	namespaces, err := utils.RunCommand("oc", "get", "namespaces", "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return fmt.Errorf("failed to get namespaces: %v", err)
	}

	for _, namespace := range strings.Split(namespaces, " ") {
		if namespace == "" {
			continue
		}

		// Create namespace metrics
		nsMetrics := &NamespaceMetrics{
			Name: namespace,
		}

		// Query for namespace CPU usage
		nsCPUQuery := fmt.Sprintf("sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{namespace='%s'})", namespace)
		nsCPUOutput, err := utils.RunCommand("oc", "exec", "-n", "openshift-monitoring", "prometheus-k8s-0", "-c", "prometheus", "--",
			"curl", "-s", fmt.Sprintf("http://localhost:9090/api/v1/query?query=%s", nsCPUQuery))

		if err == nil && strings.Contains(nsCPUOutput, "\"value\"") {
			parts := strings.Split(nsCPUOutput, "\"value\":[")
			if len(parts) > 1 {
				valueStr := strings.Split(strings.Split(parts[1], "]")[0], ",")[1]
				valueStr = strings.Trim(valueStr, "\" ")
				cpuUsage, err := strconv.ParseFloat(valueStr, 64)
				if err == nil {
					nsMetrics.CPUUsage = cpuUsage
				}
			}
		}

		// Query for namespace memory usage
		nsMemQuery := fmt.Sprintf("sum(container_memory_working_set_bytes{namespace='%s'})", namespace)
		nsMemOutput, err := utils.RunCommand("oc", "exec", "-n", "openshift-monitoring", "prometheus-k8s-0", "-c", "prometheus", "--",
			"curl", "-s", fmt.Sprintf("http://localhost:9090/api/v1/query?query=%s", nsMemQuery))

		if err == nil && strings.Contains(nsMemOutput, "\"value\"") {
			parts := strings.Split(nsMemOutput, "\"value\":[")
			if len(parts) > 1 {
				valueStr := strings.Split(strings.Split(parts[1], "]")[0], ",")[1]
				valueStr = strings.Trim(valueStr, "\" ")
				memUsage, err := strconv.ParseFloat(valueStr, 64)
				if err == nil {
					nsMetrics.MemoryUsage = memUsage
				}
			}
		}

		// Get requests and limits
		resourceInfo, err := utils.RunCommand("oc", "get", "pods", "-n", namespace, "-o", "jsonpath={range .items[*].spec.containers[*]}{.resources.requests.cpu},{.resources.limits.cpu},{.resources.requests.memory},{.resources.limits.memory}{\"\\n\"}{end}")
		if err == nil {
			for _, line := range strings.Split(resourceInfo, "\n") {
				if line == "" {
					continue
				}

				// Parse comma-separated values
				resources := strings.Split(line, ",")
				if len(resources) == 4 {
					// CPU requests
					if resources[0] != "" {
						cpuReq, err := parseResourceQuantity(resources[0])
						if err == nil {
							nsMetrics.CPURequests += cpuReq
						}
					}

					// CPU limits
					if resources[1] != "" {
						cpuLim, err := parseResourceQuantity(resources[1])
						if err == nil {
							nsMetrics.CPULimits += cpuLim
						}
					}

					// Memory requests
					if resources[2] != "" {
						memReq, err := parseResourceQuantity(resources[2])
						if err == nil {
							nsMetrics.MemoryRequests += memReq
						}
					}

					// Memory limits
					if resources[3] != "" {
						memLim, err := parseResourceQuantity(resources[3])
						if err == nil {
							nsMetrics.MemoryLimits += memLim
						}
					}
				}
			}
		}

		// Store the namespace metrics
		metrics.NamespaceMetrics[namespace] = nsMetrics
	}

	// Get network metrics
	netRecvQuery := "sum(irate(container_network_receive_bytes_total[5m])) / 1024 / 1024"
	netTransQuery := "sum(irate(container_network_transmit_bytes_total[5m])) / 1024 / 1024"

	netRecvOutput, err := utils.RunCommand("oc", "exec", "-n", "openshift-monitoring", "prometheus-k8s-0", "-c", "prometheus", "--",
		"curl", "-s", fmt.Sprintf("http://localhost:9090/api/v1/query?query=%s", netRecvQuery))

	if err == nil && strings.Contains(netRecvOutput, "\"value\"") {
		parts := strings.Split(netRecvOutput, "\"value\":[")
		if len(parts) > 1 {
			valueStr := strings.Split(strings.Split(parts[1], "]")[0], ",")[1]
			valueStr = strings.Trim(valueStr, "\" ")
			netRecv, err := strconv.ParseFloat(valueStr, 64)
			if err == nil {
				metrics.NetworkReceiveBandwidth = netRecv
			}
		}
	}

	netTransOutput, err := utils.RunCommand("oc", "exec", "-n", "openshift-monitoring", "prometheus-k8s-0", "-c", "prometheus", "--",
		"curl", "-s", fmt.Sprintf("http://localhost:9090/api/v1/query?query=%s", netTransQuery))

	if err == nil && strings.Contains(netTransOutput, "\"value\"") {
		parts := strings.Split(netTransOutput, "\"value\":[")
		if len(parts) > 1 {
			valueStr := strings.Split(strings.Split(parts[1], "]")[0], ",")[1]
			valueStr = strings.Trim(valueStr, "\" ")
			netTrans, err := strconv.ParseFloat(valueStr, 64)
			if err == nil {
				metrics.NetworkTransmitBandwidth = netTrans
			}
		}
	}

	// Get storage metrics
	storageIOPSQuery := "sum(irate(node_disk_reads_completed_total[5m]) + irate(node_disk_writes_completed_total[5m]))"
	storageThroughputQuery := "sum(irate(node_disk_read_bytes_total[5m]) + irate(node_disk_written_bytes_total[5m])) / 1024 / 1024"

	storageIOPSOutput, err := utils.RunCommand("oc", "exec", "-n", "openshift-monitoring", "prometheus-k8s-0", "-c", "prometheus", "--",
		"curl", "-s", fmt.Sprintf("http://localhost:9090/api/v1/query?query=%s", storageIOPSQuery))

	if err == nil && strings.Contains(storageIOPSOutput, "\"value\"") {
		parts := strings.Split(storageIOPSOutput, "\"value\":[")
		if len(parts) > 1 {
			valueStr := strings.Split(strings.Split(parts[1], "]")[0], ",")[1]
			valueStr = strings.Trim(valueStr, "\" ")
			iops, err := strconv.ParseFloat(valueStr, 64)
			if err == nil {
				metrics.StorageIOPS = iops
			}
		}
	}

	storageThroughputOutput, err := utils.RunCommand("oc", "exec", "-n", "openshift-monitoring", "prometheus-k8s-0", "-c", "prometheus", "--",
		"curl", "-s", fmt.Sprintf("http://localhost:9090/api/v1/query?query=%s", storageThroughputQuery))

	if err == nil && strings.Contains(storageThroughputOutput, "\"value\"") {
		parts := strings.Split(storageThroughputOutput, "\"value\":[")
		if len(parts) > 1 {
			valueStr := strings.Split(strings.Split(parts[1], "]")[0], ",")[1]
			valueStr = strings.Trim(valueStr, "\" ")
			throughput, err := strconv.ParseFloat(valueStr, 64)
			if err == nil {
				metrics.StorageThroughput = throughput
			}
		}
	}

	return nil
}

// getMetricsFromMetricsServer tries to get metrics from the Kubernetes Metrics Server
func (c *ClusterPerformanceCheck) getMetricsFromMetricsServer(dynamicClient dynamic.Interface, metrics *PerformanceMetrics) error {
	// Define NodeMetrics GVR
	nodeMetricsGVR := schema.GroupVersionResource{
		Group:    "metrics.k8s.io",
		Version:  "v1beta1",
		Resource: "nodes",
	}

	// Define PodMetrics GVR
	podMetricsGVR := schema.GroupVersionResource{
		Group:    "metrics.k8s.io",
		Version:  "v1beta1",
		Resource: "pods",
	}

	// Get node metrics
	nodeMetricsList, err := dynamicClient.Resource(nodeMetricsGVR).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node metrics: %v", err)
	}

	// Process node metrics
	totalCPUUsage := 0.0
	totalMemoryUsage := 0.0

	for _, nm := range nodeMetricsList.Items {
		// Extract node name
		nodeName, found, err := unstructured.NestedString(nm.Object, "metadata", "name")
		if err != nil || !found {
			continue
		}

		// Create node metrics if it doesn't exist
		if _, exists := metrics.NodeMetrics[nodeName]; !exists {
			metrics.NodeMetrics[nodeName] = &NodeMetrics{
				Name: nodeName,
			}
		}

		// Extract CPU usage
		cpuUsageStr, found, err := unstructured.NestedString(nm.Object, "usage", "cpu")
		if err == nil && found {
			cpuUsage, err := parseResourceQuantity(cpuUsageStr)
			if err == nil {
				metrics.NodeMetrics[nodeName].CPUUsage = cpuUsage
				totalCPUUsage += cpuUsage
			}
		}

		// Extract memory usage
		memUsageStr, found, err := unstructured.NestedString(nm.Object, "usage", "memory")
		if err == nil && found {
			memUsage, err := parseResourceQuantity(memUsageStr)
			if err == nil {
				metrics.NodeMetrics[nodeName].MemoryUsage = memUsage
				totalMemoryUsage += memUsage
			}
		}
	}

	// Get pod metrics
	podMetricsList, err := dynamicClient.Resource(podMetricsGVR).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get pod metrics: %v", err)
	}

	// Process pod metrics
	for _, pm := range podMetricsList.Items {
		// Extract namespace
		namespace, found, err := unstructured.NestedString(pm.Object, "metadata", "namespace")
		if err != nil || !found {
			continue
		}

		// Create namespace metrics if it doesn't exist
		if _, exists := metrics.NamespaceMetrics[namespace]; !exists {
			metrics.NamespaceMetrics[namespace] = &NamespaceMetrics{
				Name: namespace,
			}
		}

		// Extract container metrics
		containers, found, err := unstructured.NestedSlice(pm.Object, "containers")
		if err != nil || !found {
			continue
		}

		// Sum up container metrics for the pod
		for _, c := range containers {
			container, ok := c.(map[string]interface{})
			if !ok {
				continue
			}

			// Extract CPU usage
			usage, found, err := unstructured.NestedMap(container, "usage")
			if err != nil || !found {
				continue
			}

			cpuUsageStr, found := usage["cpu"]
			if found {
				cpuUsage, err := parseResourceQuantity(fmt.Sprintf("%v", cpuUsageStr))
				if err == nil {
					metrics.NamespaceMetrics[namespace].CPUUsage += cpuUsage
				}
			}

			memUsageStr, found := usage["memory"]
			if found {
				memUsage, err := parseResourceQuantity(fmt.Sprintf("%v", memUsageStr))
				if err == nil {
					metrics.NamespaceMetrics[namespace].MemoryUsage += memUsage
				}
			}
		}
	}

	// Calculate overall CPU and memory utilization
	totalCPUCapacity := 0.0
	totalMemoryCapacity := 0.0

	for _, nodeMetric := range metrics.NodeMetrics {
		if nodeMetric.CPUCapacity > 0 {
			totalCPUCapacity += nodeMetric.CPUCapacity
		}

		if nodeMetric.MemoryCapacity > 0 {
			totalMemoryCapacity += nodeMetric.MemoryCapacity
		}
	}

	if totalCPUCapacity > 0 {
		metrics.CPUUtilization = (totalCPUUsage / totalCPUCapacity) * 100
	}

	if totalMemoryCapacity > 0 {
		metrics.MemoryUtilization = (totalMemoryUsage / totalMemoryCapacity) * 100
	}

	return nil
}

// getMetricsFromOCCommands uses oc commands to get metrics as a last resort
func (c *ClusterPerformanceCheck) getMetricsFromOCCommands(metrics *PerformanceMetrics) error {
	// Use "oc adm top nodes" to get node resource usage
	nodeUsageOutput, err := utils.RunCommand("oc", "adm", "top", "nodes")
	if err != nil {
		return fmt.Errorf("failed to get node usage: %v", err)
	}

	// Parse node usage output
	lines := strings.Split(nodeUsageOutput, "\n")
	if len(lines) < 2 {
		return fmt.Errorf("unexpected output from 'oc adm top nodes'")
	}

	totalCPUUsage := 0.0
	totalCPUCapacity := 0.0
	totalMemoryUsage := 0.0
	totalMemoryCapacity := 0.0

	// Start from index 1 to skip the header
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		nodeName := fields[0]

		// CPU usage (format: number% or number)
		cpuUsageStr := strings.TrimSuffix(fields[2], "%")
		cpuUsage, err := strconv.ParseFloat(cpuUsageStr, 64)
		if err != nil {
			continue
		}

		// Memory usage (format: number%)
		memUsageStr := strings.TrimSuffix(fields[4], "%")
		memUsage, err := strconv.ParseFloat(memUsageStr, 64)
		if err != nil {
			continue
		}

		// Get node capacity
		nodeCap, err := utils.RunCommand("oc", "get", "node", nodeName, "-o", "jsonpath={.status.capacity.cpu},{.status.capacity.memory}")
		if err != nil {
			continue
		}

		capacities := strings.Split(nodeCap, ",")
		if len(capacities) != 2 {
			continue
		}

		// CPU capacity
		cpuCap, err := parseResourceQuantity(capacities[0])
		if err != nil {
			continue
		}

		// Memory capacity
		memCap, err := parseResourceQuantity(capacities[1])
		if err != nil {
			continue
		}

		// Create node metrics if it doesn't exist
		if _, exists := metrics.NodeMetrics[nodeName]; !exists {
			metrics.NodeMetrics[nodeName] = &NodeMetrics{
				Name: nodeName,
			}
		}

		// Update node metrics
		metrics.NodeMetrics[nodeName].CPUCapacity = cpuCap
		metrics.NodeMetrics[nodeName].CPUUsage = (cpuUsage / 100) * cpuCap
		metrics.NodeMetrics[nodeName].MemoryCapacity = memCap
		metrics.NodeMetrics[nodeName].MemoryUsage = (memUsage / 100) * memCap

		// Update totals
		totalCPUCapacity += cpuCap
		totalCPUUsage += (cpuUsage / 100) * cpuCap
		totalMemoryCapacity += memCap
		totalMemoryUsage += (memUsage / 100) * memCap
	}

	// Calculate overall utilization
	if totalCPUCapacity > 0 {
		metrics.CPUUtilization = (totalCPUUsage / totalCPUCapacity) * 100
	}

	if totalMemoryCapacity > 0 {
		metrics.MemoryUtilization = (totalMemoryUsage / totalMemoryCapacity) * 100
	}

	// Use "oc adm top pods --all-namespaces" to get pod resource usage
	podUsageOutput, err := utils.RunCommand("oc", "adm", "top", "pods", "--all-namespaces")
	if err != nil {
		return fmt.Errorf("failed to get pod usage: %v", err)
	}

	// Parse pod usage output
	lines = strings.Split(podUsageOutput, "\n")
	if len(lines) < 2 {
		return fmt.Errorf("unexpected output from 'oc adm top pods'")
	}

	// Start from index 1 to skip the header
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		namespace := fields[0]
		podName := fields[1]

		// CPU usage (format: number or numberm)
		cpuUsage, err := parseResourceQuantity(fields[2])
		if err != nil {
			continue
		}

		// Memory usage (format: number or numberMi)
		memUsage, err := parseResourceQuantity(fields[3])
		if err != nil {
			continue
		}

		// Create namespace metrics if it doesn't exist
		if _, exists := metrics.NamespaceMetrics[namespace]; !exists {
			metrics.NamespaceMetrics[namespace] = &NamespaceMetrics{
				Name: namespace,
			}
		}

		// Update namespace metrics
		metrics.NamespaceMetrics[namespace].CPUUsage += cpuUsage
		metrics.NamespaceMetrics[namespace].MemoryUsage += memUsage

		// Get pod resource requests and limits
		podResources, err := utils.RunCommand("oc", "get", "pod", podName, "-n", namespace, "-o", "jsonpath={range .spec.containers[*]}{.resources.requests.cpu},{.resources.limits.cpu},{.resources.requests.memory},{.resources.limits.memory}{\"\\n\"}{end}")
		if err != nil {
			continue
		}

		for _, resourceLine := range strings.Split(podResources, "\n") {
			if resourceLine == "" {
				continue
			}

			resources := strings.Split(resourceLine, ",")
			if len(resources) != 4 {
				continue
			}

			// CPU requests
			if resources[0] != "" {
				cpuReq, err := parseResourceQuantity(resources[0])
				if err == nil {
					metrics.NamespaceMetrics[namespace].CPURequests += cpuReq
				}
			}

			// CPU limits
			if resources[1] != "" {
				cpuLim, err := parseResourceQuantity(resources[1])
				if err == nil {
					metrics.NamespaceMetrics[namespace].CPULimits += cpuLim
				}
			}

			// Memory requests
			if resources[2] != "" {
				memReq, err := parseResourceQuantity(resources[2])
				if err == nil {
					metrics.NamespaceMetrics[namespace].MemoryRequests += memReq
				}
			}

			// Memory limits
			if resources[3] != "" {
				memLim, err := parseResourceQuantity(resources[3])
				if err == nil {
					metrics.NamespaceMetrics[namespace].MemoryLimits += memLim
				}
			}
		}
	}

	return nil
}

// processNodeInformation processes node information to calculate capacities
func (c *ClusterPerformanceCheck) processNodeInformation(nodes []corev1.Node, metrics *PerformanceMetrics) {
	for _, node := range nodes {
		// Create node metrics if it doesn't exist
		if _, exists := metrics.NodeMetrics[node.Name]; !exists {
			metrics.NodeMetrics[node.Name] = &NodeMetrics{
				Name: node.Name,
			}
		}

		// Get CPU capacity and allocatable
		cpuCap := parseQuantity(node.Status.Capacity.Cpu())
		cpuAlloc := parseQuantity(node.Status.Allocatable.Cpu())

		// Get memory capacity and allocatable
		memCap := parseQuantity(node.Status.Capacity.Memory())
		memAlloc := parseQuantity(node.Status.Allocatable.Memory())

		// Update node metrics if not already set
		nodeMetric := metrics.NodeMetrics[node.Name]

		if nodeMetric.CPUCapacity == 0 {
			nodeMetric.CPUCapacity = cpuCap
		}

		if nodeMetric.CPUAllocatable == 0 {
			nodeMetric.CPUAllocatable = cpuAlloc
		}

		if nodeMetric.MemoryCapacity == 0 {
			nodeMetric.MemoryCapacity = memCap
		}

		if nodeMetric.MemoryAllocatable == 0 {
			nodeMetric.MemoryAllocatable = memAlloc
		}
	}
}

// calculateCommitmentRatios calculates CPU and memory commitment ratios
func (c *ClusterPerformanceCheck) calculateCommitmentRatios(metrics *PerformanceMetrics) {
	totalCPUCapacity := 0.0
	totalMemoryCapacity := 0.0
	totalCPURequests := 0.0
	totalCPULimits := 0.0
	totalMemoryRequests := 0.0
	totalMemoryLimits := 0.0

	// Sum up total capacity
	for _, nodeMetric := range metrics.NodeMetrics {
		totalCPUCapacity += nodeMetric.CPUAllocatable
		totalMemoryCapacity += nodeMetric.MemoryAllocatable
	}

	// Sum up total requests and limits
	for _, nsMetric := range metrics.NamespaceMetrics {
		totalCPURequests += nsMetric.CPURequests
		totalCPULimits += nsMetric.CPULimits
		totalMemoryRequests += nsMetric.MemoryRequests
		totalMemoryLimits += nsMetric.MemoryLimits
	}

	// Calculate commitment ratios
	if totalCPUCapacity > 0 {
		metrics.CPURequestsCommitment = (totalCPURequests / totalCPUCapacity) * 100
		metrics.CPULimitsCommitment = (totalCPULimits / totalCPUCapacity) * 100
	}

	if totalMemoryCapacity > 0 {
		metrics.MemoryRequestsCommitment = (totalMemoryRequests / totalMemoryCapacity) * 100
		metrics.MemoryLimitsCommitment = (totalMemoryLimits / totalMemoryCapacity) * 100
	}
}

// getTopNamespaces returns the top n namespaces by CPU and memory usage
func (c *ClusterPerformanceCheck) getTopNamespaces(metrics *PerformanceMetrics, n int) ([]string, []string) {
	// Create namespace slices by CPU and memory usage
	type nsUsage struct {
		name  string
		usage float64
	}

	var cpuUsages []nsUsage
	var memUsages []nsUsage

	for name, nsMetric := range metrics.NamespaceMetrics {
		cpuUsages = append(cpuUsages, nsUsage{name: name, usage: nsMetric.CPUUsage})
		memUsages = append(memUsages, nsUsage{name: name, usage: nsMetric.MemoryUsage})
	}

	// Sort by CPU usage (descending)
	for i := 0; i < len(cpuUsages)-1; i++ {
		for j := i + 1; j < len(cpuUsages); j++ {
			if cpuUsages[i].usage < cpuUsages[j].usage {
				cpuUsages[i], cpuUsages[j] = cpuUsages[j], cpuUsages[i]
			}
		}
	}

	// Sort by memory usage (descending)
	for i := 0; i < len(memUsages)-1; i++ {
		for j := i + 1; j < len(memUsages); j++ {
			if memUsages[i].usage < memUsages[j].usage {
				memUsages[i], memUsages[j] = memUsages[j], memUsages[i]
			}
		}
	}

	// Take top n namespaces
	topCPU := make([]string, 0, n)
	topMem := make([]string, 0, n)

	for i := 0; i < len(cpuUsages) && i < n; i++ {
		topCPU = append(topCPU, cpuUsages[i].name)
	}

	for i := 0; i < len(memUsages) && i < n; i++ {
		topMem = append(topMem, memUsages[i].name)
	}

	return topCPU, topMem
}

// Helper functions

// parseQuantity parses a resource.Quantity and returns its value in float64
func parseQuantity(quantity *resource.Quantity) float64 {
	if quantity == nil {
		return 0
	}

	// Extract value based on resource type
	if quantity.IsZero() {
		return 0
	}

	// For CPU resources
	if quantity.Format == resource.DecimalSI {
		// For CPU, directly get the value in cores
		return float64(quantity.MilliValue()) / 1000.0
	}

	// For memory resources (in bytes)
	return float64(quantity.Value())
}

// parseResourceQuantity parses a resource quantity string and returns its value in float64
func parseResourceQuantity(quantity string) (float64, error) {
	// Handle CPU formats like "100m" or "0.1"
	if strings.HasSuffix(quantity, "m") {
		value, err := strconv.ParseFloat(strings.TrimSuffix(quantity, "m"), 64)
		if err != nil {
			return 0, err
		}
		return value / 1000, nil
	}

	// Handle memory formats
	memoryMultipliers := map[string]float64{
		"Ki": 1024,
		"Mi": 1024 * 1024,
		"Gi": 1024 * 1024 * 1024,
		"Ti": 1024 * 1024 * 1024 * 1024,
		"Pi": 1024 * 1024 * 1024 * 1024 * 1024,
		"Ei": 1024 * 1024 * 1024 * 1024 * 1024 * 1024,
		"K":  1000,
		"M":  1000 * 1000,
		"G":  1000 * 1000 * 1000,
		"T":  1000 * 1000 * 1000 * 1000,
		"P":  1000 * 1000 * 1000 * 1000 * 1000,
		"E":  1000 * 1000 * 1000 * 1000 * 1000 * 1000,
	}

	for suffix, multiplier := range memoryMultipliers {
		if strings.HasSuffix(quantity, suffix) {
			value, err := strconv.ParseFloat(strings.TrimSuffix(quantity, suffix), 64)
			if err != nil {
				return 0, err
			}
			return value * multiplier, nil
		}
	}

	// Default case, just parse as float
	return strconv.ParseFloat(quantity, 64)
}

// formatBytes formats bytes in human-readable format
func formatBytes(bytes float64) string {
	const unit = 1024.0
	if bytes < unit {
		return fmt.Sprintf("%.2f B", bytes)
	}

	div, exp := unit, 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.2f %ciB", bytes/div, "KMGTPE"[exp])
}
