/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-17

This file acts as a provider for cluster-related health checks. It includes:

- A registry of all available cluster configuration health checks
- Functions to retrieve and initialize cluster checks
- Organization of checks related to node configuration, operators, and cluster version
- Registration of checks for core cluster components

The provider ensures that all cluster-related health checks are properly registered and available for execution by the main runner.
*/

package cluster

import (
	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all cluster-related health checks
func GetChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Following the order in the PDF:

	// Infrastructure Provider - first in Infra category
	checks = append(checks, NewInfrastructureProviderCheck())

	// Installation Type
	checks = append(checks, NewInstallationTypeCheck())

	// UPI with MachineSets check
	checks = append(checks, NewUPIMachineSetCheck())

	// Node Status
	checks = append(checks, NewNodeStatusCheck())

	// Node Usage
	checks = append(checks, NewNodeUsageCheck())

	// Cluster Version - first in Cluster Config
	checks = append(checks, NewClusterVersionCheck())

	// Cluster Operators
	checks = append(checks, NewClusterOperatorsCheck())

	// Control Nodes Schedulable
	checks = append(checks, NewControlNodeSchedulableCheck())

	// Infrastructure Nodes
	checks = append(checks, NewInfrastructureNodesCheck())

	// Workload off Infra Nodes
	checks = append(checks, NewWorkloadOffInfraNodesCheck())

	// Default Project Template - comes from security in PDF

	// Default Node Schedule
	checks = append(checks, NewDefaultNodeScheduleCheck()) // Fixed function name here

	// Infra machine config pool
	checks = append(checks, NewInfraMachineConfigPoolCheck())

	// Kubelet Configuration
	checks = append(checks, NewKubeletGarbageCollectionCheck())

	// ETCD health and performance checks
	checks = append(checks, NewEtcdHealthCheck())
	checks = append(checks, NewEtcdPerformanceCheck())

	// Openshift Proxy Settings
	checks = append(checks, NewProxySettingsCheck())

	// Infrastructure node taints
	checks = append(checks, NewInfraTaintsCheck())

	return checks
}
