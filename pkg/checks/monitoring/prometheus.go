package monitoring

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// PrometheusCheck checks if Prometheus is running correctly
type PrometheusCheck struct {
	healthcheck.BaseCheck
}

// NewPrometheusCheck creates a new Prometheus check
func NewPrometheusCheck() *PrometheusCheck {
	return &PrometheusCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"prometheus",
			"Prometheus",
			"Checks if Prometheus is running correctly",
			healthcheck.CategoryMonitoring,
		),
	}
}

// Run executes the health check
func (c *PrometheusCheck) Run() (healthcheck.Result, error) {
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

	// Create a dynamic client
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to create Kubernetes client",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error creating Kubernetes client: %v", err)
	}

	// Check for Prometheus pods in the openshift-monitoring namespace
	clientset, err := utils.GetClientSet()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get Kubernetes client",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting Kubernetes client: %v", err)
	}

	// Check if the monitoring operator is installed
	ctx := context.Background()
	promPods, err := clientset.CoreV1().Pods("openshift-monitoring").List(ctx, metav1.ListOptions{
		LabelSelector: "app=prometheus",
	})
	if err != nil {
		// Check if the namespace exists
		_, nsErr := clientset.CoreV1().Namespaces().Get(ctx, "openshift-monitoring", metav1.GetOptions{})
		if nsErr != nil {
			return healthcheck.NewResult(
				c.ID(),
				healthcheck.StatusCritical,
				"The openshift-monitoring namespace does not exist",
				healthcheck.ResultKeyRequired,
			), fmt.Errorf("error: openshift-monitoring namespace not found: %v", nsErr)
		}

		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check for Prometheus pods",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking for Prometheus pods: %v", err)
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-monitoring", "-l", "app=prometheus")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed Prometheus pod information"
	}

	// Check for Prometheus operands using the PrometheusRule CRD
	// This is a good way to check if Prometheus is properly installed
	prometheusRules := schema.GroupVersionResource{
		Group:    "monitoring.coreos.com",
		Version:  "v1",
		Resource: "prometheusrules",
	}

	_, err = client.Resource(prometheusRules).List(ctx, metav1.ListOptions{})
	hasPrometheusRules := err == nil

	// Check for alerts
	alertsOut, err := utils.RunCommandWithTimeout(10, "oc", "exec",
		"-n", "openshift-monitoring", "prometheus-k8s-0", "-c", "prometheus",
		"--", "curl", "-s", "http://localhost:9090/api/v1/alerts")
	hasAlertsAccess := err == nil

	// Get current firing alerts if possible
	var firingAlerts []string
	if hasAlertsAccess && strings.Contains(alertsOut, "\"alerts\"") && strings.Contains(alertsOut, "\"firing\"") {
		// This is a very simplistic parsing of alerts - in a real implementation, proper JSON parsing would be used
		alertLines := strings.Split(alertsOut, "\"name\":\"")
		for _, line := range alertLines[1:] {
			if strings.Contains(line, "\"state\":\"firing\"") {
				parts := strings.Split(line, "\"")
				if len(parts) > 0 {
					firingAlerts = append(firingAlerts, parts[0])
				}
			}
		}
	}

	// Determine the status of Prometheus
	if len(promPods.Items) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"No Prometheus pods found in the openshift-monitoring namespace",
			healthcheck.ResultKeyRequired,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// Check if all pods are ready
	allReady := true
	for _, pod := range promPods.Items {
		if pod.Status.Phase != "Running" {
			allReady = false
			break
		}
	}

	if !allReady {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Not all Prometheus pods are running",
			healthcheck.ResultKeyRequired,
		)
		result.Detail = detailedOut
		return result, nil
	}

	if !hasPrometheusRules {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"Prometheus is running but PrometheusRules CRD is not available",
			healthcheck.ResultKeyRecommended,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// Generate the result
	var resultStatus healthcheck.Status
	var resultMessage string
	var resultKey healthcheck.ResultKey

	if len(firingAlerts) > 0 {
		resultStatus = healthcheck.StatusWarning
		resultMessage = fmt.Sprintf("Prometheus is running but there are %d firing alerts", len(firingAlerts))
		resultKey = healthcheck.ResultKeyAdvisory
	} else {
		resultStatus = healthcheck.StatusOK
		resultMessage = "Prometheus is running correctly"
		resultKey = healthcheck.ResultKeyNoChange
	}

	result := healthcheck.NewResult(
		c.ID(),
		resultStatus,
		resultMessage,
		resultKey,
	)

	// Add details about firing alerts if any
	if len(firingAlerts) > 0 {
		detail := fmt.Sprintf("Firing alerts:\n%s\n\n%s", strings.Join(firingAlerts, "\n"), detailedOut)
		result.Detail = detail
		result.AddRecommendation("Investigate the firing alerts in the Prometheus UI")
	} else {
		result.Detail = detailedOut
	}

	return result, nil
}

// AlertManagerCheck checks if AlertManager is configured correctly
type AlertManagerCheck struct {
	healthcheck.BaseCheck
}

// NewAlertManagerCheck creates a new AlertManager check
func NewAlertManagerCheck() *AlertManagerCheck {
	return &AlertManagerCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"alertmanager",
			"AlertManager",
			"Checks if AlertManager is configured correctly",
			healthcheck.CategoryMonitoring,
		),
	}
}

// Run executes the health check
func (c *AlertManagerCheck) Run() (healthcheck.Result, error) {
	// Check for AlertManager pods in the openshift-monitoring namespace
	clientset, err := utils.GetClientSet()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get Kubernetes client",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting Kubernetes client: %v", err)
	}

	// Check if AlertManager is running
	ctx := context.Background()
	alertManagerPods, err := clientset.CoreV1().Pods("openshift-monitoring").List(ctx, metav1.ListOptions{
		LabelSelector: "app=alertmanager",
	})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to check for AlertManager pods",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error checking for AlertManager pods: %v", err)
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-monitoring", "-l", "app=alertmanager")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed AlertManager pod information"
	}

	// Check if AlertManager configuration exists
	configOut, err := utils.RunCommand("oc", "get", "secret", "alertmanager-main", "-n", "openshift-monitoring")
	hasConfig := err == nil

	// Check if at least one receiver is configured (simplified check)
	receiversOut, err := utils.RunCommand("oc", "get", "secret", "alertmanager-main", "-n", "openshift-monitoring", "-o", "jsonpath={.data.alertmanager\\.yaml}")
	hasReceivers := err == nil && strings.Contains(receiversOut, "receivers")

	// Check if all pods are ready
	allReady := true
	for _, pod := range alertManagerPods.Items {
		if pod.Status.Phase != "Running" {
			allReady = false
			break
		}
	}

	// Determine the status of AlertManager
	if len(alertManagerPods.Items) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"No AlertManager pods found in the openshift-monitoring namespace",
			healthcheck.ResultKeyRequired,
		)
		result.Detail = detailedOut
		return result, nil
	}

	if !allReady {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Not all AlertManager pods are running",
			healthcheck.ResultKeyRequired,
		)
		result.Detail = detailedOut
		return result, nil
	}

	if !hasConfig {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"AlertManager is running but configuration is missing",
			healthcheck.ResultKeyRecommended,
		)
		result.Detail = detailedOut
		return result, nil
	}

	if !hasReceivers {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"AlertManager is running but no alert receivers are configured",
			healthcheck.ResultKeyRecommended,
		)

		result.AddRecommendation("Configure alert receivers to ensure alerts are sent to the appropriate channels")
		result.Detail = detailedOut

		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		"AlertManager is running and configured correctly",
		healthcheck.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}

// GrafanaCheck checks if Grafana is running correctly
type GrafanaCheck struct {
	healthcheck.BaseCheck
}

// NewGrafanaCheck creates a new Grafana check
func NewGrafanaCheck() *GrafanaCheck {
	return &GrafanaCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"grafana",
			"Grafana",
			"Checks if Grafana is running correctly",
			healthcheck.CategoryMonitoring,
		),
	}
}

// Run executes the health check
func (c *GrafanaCheck) Run() (healthcheck.Result, error) {
	// Check for Grafana pods in the openshift-monitoring namespace
	clientset, err := utils.GetClientSet()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get Kubernetes client",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting Kubernetes client: %v", err)
	}

	// Check if Grafana is running
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	grafanaPods, err := clientset.CoreV1().Pods("openshift-monitoring").List(ctx, metav1.ListOptions{
		LabelSelector: "app=grafana",
	})
	if err != nil {
		// Check the OpenShift console plugin instead
		_, pluginErr := utils.RunCommand("oc", "get", "consoleplugin", "monitoring-plugin")
		if pluginErr == nil {
			// The monitoring plugin exists, which means Grafana is integrated with the console
			return healthcheck.NewResult(
				c.ID(),
				healthcheck.StatusOK,
				"Grafana functionality is available through the OpenShift console plugin",
				healthcheck.ResultKeyNoChange,
			), nil
		}

		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"Failed to check for Grafana pods and console plugin not found",
			healthcheck.ResultKeyAdvisory,
		), fmt.Errorf("error checking for Grafana: %v", err)
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-monitoring", "-l", "app=grafana")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed Grafana pod information"
	}

	// Determine the status of Grafana
	if len(grafanaPods.Items) == 0 {
		// Check the OpenShift console plugin instead
		_, pluginErr := utils.RunCommand("oc", "get", "consoleplugin", "monitoring-plugin")
		if pluginErr == nil {
			// The monitoring plugin exists, which means Grafana is integrated with the console
			return healthcheck.NewResult(
				c.ID(),
				healthcheck.StatusOK,
				"Grafana functionality is available through the OpenShift console plugin",
				healthcheck.ResultKeyNoChange,
			), nil
		}

		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"No Grafana pods found in the openshift-monitoring namespace and console plugin not found",
			healthcheck.ResultKeyAdvisory,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// Check if all pods are ready
	allReady := true
	for _, pod := range grafanaPods.Items {
		if pod.Status.Phase != "Running" {
			allReady = false
			break
		}
	}

	if !allReady {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			"Not all Grafana pods are running",
			healthcheck.ResultKeyAdvisory,
		)
		result.Detail = detailedOut
		return result, nil
	}

	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		"Grafana is running correctly",
		healthcheck.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
