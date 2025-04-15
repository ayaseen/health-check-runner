/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for user workload monitoring. It:

- Verifies if monitoring for user-defined projects is enabled
- Checks the cluster-monitoring-config ConfigMap for proper settings
- Examines the existence of the openshift-user-workload-monitoring namespace
- Provides recommendations for enabling application monitoring
- Helps ensure proper visibility into application performance

This check helps administrators configure proper monitoring for application workloads beyond the default system monitoring.
*/

package monitoring

import (
	"context"
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// UserWorkloadMonitoringCheck checks if user-defined workload monitoring is enabled
type UserWorkloadMonitoringCheck struct {
	healthcheck.BaseCheck
}

// NewUserWorkloadMonitoringCheck creates a new user workload monitoring check
func NewUserWorkloadMonitoringCheck() *UserWorkloadMonitoringCheck {
	return &UserWorkloadMonitoringCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"user-workload-monitoring",
			"User Workload Monitoring",
			"Checks if monitoring for user-defined projects is enabled",
			types.CategoryOpReady,
		),
	}
}

// Run executes the health check
func (c *UserWorkloadMonitoringCheck) Run() (healthcheck.Result, error) {
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

	// Check if the openshift-monitoring namespace exists
	_, err = clientset.CoreV1().Namespaces().Get(context.TODO(), "openshift-monitoring", metav1.GetOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get openshift-monitoring namespace",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting openshift-monitoring namespace: %v", err)
	}

	// Check if the cluster-monitoring-config ConfigMap exists
	cm, err := clientset.CoreV1().ConfigMaps("openshift-monitoring").Get(context.TODO(), "cluster-monitoring-config", metav1.GetOptions{})

	var userWorkloadEnabled bool
	var configMapExists bool

	if err != nil {
		configMapExists = false
		userWorkloadEnabled = false
	} else {
		configMapExists = true

		// Check if the key exists
		data, ok := cm.Data["config.yaml"]
		if ok {
			var configData ConfigData
			if err := yaml.Unmarshal([]byte(data), &configData); err == nil {
				userWorkloadEnabled = configData.EnableUserWorkload
			}
		}
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "configmap", "cluster-monitoring-config", "-n", "openshift-monitoring", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed monitoring ConfigMap information"
	}

	// Get the OpenShift version for recommendations
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Additional check: see if the user-workload-monitoring namespace exists
	_, nsErr := clientset.CoreV1().Namespaces().Get(context.TODO(), "openshift-user-workload-monitoring", metav1.GetOptions{})
	userWorkloadNamespaceExists := nsErr == nil

	// If both checks indicate user workload monitoring is enabled
	if (configMapExists && userWorkloadEnabled) || userWorkloadNamespaceExists {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"User workload monitoring is enabled",
			types.ResultKeyNoChange,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// User workload monitoring is not enabled
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		"User workload monitoring is not enabled",
		types.ResultKeyRecommended,
	)

	result.AddRecommendation("Enable monitoring for user-defined projects")
	result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/configuring-the-monitoring-stack", version))
	result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/monitoring/index#enabling-monitoring-for-user-defined-projects", version))

	result.Detail = detailedOut
	return result, nil
}
