package cluster

import (
	"gitlab.consulting.redhat.com/meta/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all cluster-related health checks
func GetChecks() []healthcheck.Check {
	// Retrieve the latest OpenShift version
	latestVersion := getLatestOpenShiftVersion()

	var checks []healthcheck.Check

	// Add cluster version check
	checks = append(checks, NewClusterVersionCheck(latestVersion))

	// Add cluster operators check
	checks = append(checks, NewClusterOperatorsCheck())

	// Add nodes check
	checks = append(checks, NewNodeStatusCheck())

	// Add node usage check
	checks = append(checks, NewNodeUsageCheck())

	// Add control node schedulable check
	checks = append(checks, NewControlNodeSchedulableCheck())

	// Add infrastructure nodes check
	checks = append(checks, NewInfrastructureNodesCheck())

	// Add infrastructure machine config pool check
	checks = append(checks, NewInfraMachineConfigPoolCheck())

	// Add cluster default SCC check
	checks = append(checks, NewClusterDefaultSCCCheck())

	// Add infrastructure provider check
	checks = append(checks, NewInfrastructureProviderCheck())

	// Add installation type check
	checks = append(checks, NewInstallationTypeCheck())

	// Add default project template check
	checks = append(checks, NewDefaultProjectTemplateCheck())

	// Add default node selector check
	checks = append(checks, NewDefaultNodeSelectorCheck())

	// Add kubeadmin user check
	checks = append(checks, NewKubeadminUserCheck())

	// Add workload off infra nodes check
	checks = append(checks, NewWorkloadOffInfraNodesCheck())

	return checks
}

// getLatestOpenShiftVersion returns the latest OpenShift version
func getLatestOpenShiftVersion() string {
	// This would ideally check the Red Hat site for the latest version
	// For now, we'll return a hardcoded version
	return "4.14.0"
}
